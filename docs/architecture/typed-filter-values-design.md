# Typed Filter Values — Technical Design

## 1. Executive summary

Specifica di progettazione sul branch `json-filter-support`, commit `82e7be3a1939340cfc3374cf9f8458a63f21a43b`, aggiornata il 24 ottobre 2026 alle decisioni funzionali definitive. La baseline è [current-filters-spec.md](current-filters-spec.md): il commit corrente coincide con quello documentato. La revisione riutilizza l'analisi precedente e verifica il sorgente nei punti necessari a correggerla. Modifica soltanto il presente documento; non implementa funzionalità e non progetta Balanced Filters.

La modifica è fattibile mantenendo l'architettura generale: `core.Filter` resta l'albero pubblico, `state.Where` resta l'albero interno, `filters.Applies` resta il valutatore per record e `warehouses.Expr` resta l'AST SQL. Cambiano il contratto degli operandi e i punti che lo interpretano.

Le modifiche indispensabili sono:

- Ricevere `FilterCondition.Values []json.Value` senza schema; in `core` verificare il kind JSON, risolvere lo schema e produrre una sola volta gli operandi Go.
- Usare JSON Number per tutti gli interi signed/unsigned, indipendentemente dalla bit size, e per `year`, sempre con sintassi intera e parsing esatto. Conservare la validazione corrente dei float; applicare 76/37 solo a decimal e Number di proprietà JSON, senza imporre P,S della proprietà alle soglie decimal.
- Per le proprietà `json` conservare `json.Value` anche in `state.Where` e nell'AST warehouse; eliminare le due definizioni di `JSONConditionValue` e il relativo adattatore.
- Confrontare Number con Number attraverso `decimal.Decimal` e String con String. `100`, `100.0` e `1e2` sono uguali numericamente; `"100"` appartiene a un dominio diverso.
- Consentire Boolean e Null soltanto nella membership JSON, anche in liste miste con String e Number. Un JSON Null presente può appartenere alla lista; una proprietà assente non vi appartiene mai. Applicare le regole definitive A1–A6 di §5 e §15, inclusa la String vuota esplicitamente quotata.
- Adeguare persistenza, conversione inversa all'API, costruttori Go, fixture e tutti i percorsi HTTP.
- Separare nell'Admin i valori editoriali dai valori pronti per l'API, conservando il testo dei numeri e utilizzando la serializzazione esatta già disponibile attraverso `json-bigint`.

Le difficoltà principali sono trasversali: il ramo SQL già esistente per `json.Value` significa oggi uguaglianza JSON strutturale, anche per scalari, e richiede un discriminante esplicito nell'AST per preservarlo; l'Admin usa `JSONbig` nei body ma `JSON.stringify` per alcuni filtri in query string; l'editor perde oggi la distinzione fra testo numerico quotato e non quotato. Inoltre Snowflake può aver già arrotondato i numeri memorizzati: la precisione dell'operando non recupera quella persa nello storage.

Il decoder degli operandi sarà specifico dei filtri, prendendo `types.Decode` come riferimento e riusandone i parser opportuni. Richiederà Number per tutti gli interi senza dipendere dall'adeguamento separato del sistema generale dei tipi. Le espressioni SQL ordinarie e la semantica SQL dei datetime restano quelle attuali; si corregge soltanto quanto necessario a conservare i valori tipizzati nella serializzazione. Nessun convertitore o sistema di compatibilità dei filtri precedenti è progettato qui: §11 conserva soltanto un'avvertenza per il futuro rilascio. Restano verifiche tecniche, elencate in §15, non decisioni funzionali aperte.

**Metodo ed evidenze.** **C** indica implementazione o test esistente letto; **D** una conseguenza dedotta; **P** un intervento tecnico proposto. A1–A6 identificano decisioni funzionali definitive. I riferimenti indicano file, simbolo e linea iniziale. Nell'analisi precedente sono passati test unitari esistenti selezionati in sette package Go ed è stato eseguito un esperimento Node in memoria sul vendor: risultati e limiti sono in §13.7 e §9.4. In questa revisione sono stati consultati i sorgenti pertinenti senza rieseguire quelle prove. È stata aggiunta soltanto una verifica mirata in memoria del vendor Admin sugli interi a 64 bit (§9.4). Non sono state eseguite suite Admin o prove di integrazione con PostgreSQL/Snowflake. I test passati verificano la baseline, non la funzionalità proposta.

## 2. Architettura attuale rilevante

### 2.1 Percorso degli operandi

```text
HTTP JSON → cmd → core.Filter ([]string)
                   → validateFilter(filter, schema)
                   → convertFilterToWhere(filter, schema)
                   → state.Where ([]any)
                      ├─ filters.Applies(record)
                      ├─ datastore.convertWhere → warehouses.Expr → SQL
                      └─ MarshalWhere → pipelines.filter → unmarshalWhere(InSchema)
                            → convertWhereToFilter → risposta API
```

**C.** `Filter.UnmarshalJSON` usa un decoder ricorsivo con controllo strutturale, indipendente dallo schema; `FilterCondition` ha un marshal personalizzato, ma non un proprio unmarshal personalizzato. Il decoder dell'albero legge attualmente `values` come `[]string`. La validazione controlla schema, operatore, cardinalità e valori; il convertitore successivo ripete molti parsing e ne ignora gli errori perché presume il filtro già valido. Riferimenti: [modello e metodi](../../core/filters.go#L33), [FilterCondition](../../core/filters.go#L101), [convertFilterToWhere](../../core/filters.go#L192), [unmarshalFilterRule](../../core/filters.go#L596), [validateFilter](../../core/filters.go#L730).

**C.** L'albero deve avere una radice gruppo, gruppi non vuoti, profondità massima 4 e al massimo 100 regole discendenti. Non esiste un nodo NOT generico. Gli otto operatori senza valori richiedono `Values == nil`: omissione e `values: null` sono ammessi; `values: []` è rifiutato. Con l'attuale `[]string`, un elemento `null` può diventare la stringa vuota: questo comportamento non va perpetuato nel nuovo modello. Riferimenti: [limiti](../../core/filters.go#L27), [cardinalità](../../core/filters.go#L875), [test unmarshal valori](../../core/filters_test.go#L398), [gruppi](../../core/filters.go#L961).

**C.** `state.WhereCondition.Values` è già `[]any`; il formato persistito usa già `[]json.Value` durante la decodifica, ricostruendo poi i tipi dallo schema. L'API e il formato interno differiscono per operatori e percorsi. `pipelines.filter` è `jsonb` nel database di Krenalis; le notifiche replicano anche questo payload. Riferimenti: [WhereCondition](../../core/internal/state/where.go#L98), [whereConditionJSON](../../core/internal/state/where.go#L230), [MarshalWhere](../../core/internal/state/where.go#L238), [schema pipelines](../../core/internal/initdb/schema.sql#L190), [load](../../core/internal/state/load.go#L608), [keeper creazione](../../core/internal/state/keep.go#L549).

### 2.2 Ambiguità attuale e differenze da non confondere

**C.** Un operando JSON testuale `"100"` diventa `JSONConditionValue{String: "100", Number: ...}` se è interpretabile come decimale 76/37. Il runtime e i renderer scelgono il confronto dal kind del **dato**, disponendo contemporaneamente della soglia testuale e numerica. Pertanto `properties.amount > "100"` può accettare sia il numero `150` sia la stringa `"9"`, perché `"9" > "100"` lessicograficamente. Non converte la stringa del record nel numero 9. Riferimenti: [conversione JSON](../../core/filters.go#L247), [confronto Go](../../core/internal/filters/applies.go#L186), [PG](../../warehouses/postgresql/renderexpr.go#L256), [Snowflake](../../warehouses/snowflake/renderexpr.go#L243).

**C.** Per JSON i predicati SQL sono già costruiti come booleani a due valori, con `COALESCE(..., FALSE)` e negazione esterna. Le differenze preesistenti fra Go e SQL sui valori ordinari assenti/null non sono risolte dalla tipizzazione degli operandi. La modifica deve preservarle senza estenderle al nuovo ramo JSON. Riferimenti: [dispatch Go](../../core/internal/filters/applies.go#L44), [PG JSON](../../warehouses/postgresql/renderexpr.go#L272), [Snowflake JSON](../../warehouses/snowflake/renderexpr.go#L260), [baseline, semantica e limiti](current-filters-spec.md).

### 2.3 Rappresentazioni canoniche effettive

**C.** Il commento di `types.Decode` non basta a stabilire il formato: dichiara `decimal` come String, ma il decoder richiede Number e il marshal emette Number. I test confermano il comportamento. Per `int(64)` il sistema dei dati ordinari usa invece String, mentre gli altri interi usano Number. Riferimenti: [commento](../../tools/types/value.go#L39), [decodifica tipi](../../tools/types/value.go#L463), [decimal](../../tools/types/value.go#L518), [marshal](../../tools/types/value.go#L635), [fixture Decode](../../tools/types/value_test.go#L166), [fixture Marshal](../../tools/types/value_test.go#L415).

**Decisione definitiva.** Tutti gli operandi dei filtri su proprietà intere signed/unsigned, da 8 a 64 bit, e su `year` usano JSON Number. La precedente prescrizione String per int64 è revocata. `state.Where` conserva `int` / `uint`; il decoder specifico dei filtri accetta Number anche dove `types.Decode` richiede oggi String. Questa divergenza è temporanea e intenzionale: il formato generale degli interi a 64 bit sarà adeguato in una PR separata, senza essere un prerequisito per Typed Filter Values. Questa attività non modifica `types.Decode`, `types.Marshal` o il sistema generale dei tipi.

## 3. Nuovo modello degli operandi tipizzati

### 3.1 Modello pubblico e JSON

```go
// FilterCondition represents a single filter condition.
type FilterCondition struct {
    Property string         `json:"property"`
    Operator FilterOperator `json:"operator"`
    Values   []json.Value   `json:"values,omitzero"`
}
```

`json` indica `github.com/krenalis/krenalis/tools/json`. Il tipo è una slice di byte che conserva un valore JSON, non `any` decodificato attraverso `float64`. [Value](../../tools/json/value.go#L40), [UnmarshalJSON](../../tools/json/value.go#L397).

Esempio con `age: int(32)`, `account: int(64)`, `ratio: float(64)`, `credit: decimal(5,2)` e `properties: json`:

```json
{
  "operator": "and",
  "rules": [
    {"property": "age", "operator": "is greater than", "values": [18]},
    {"property": "account", "operator": "is", "values": [9223372036854775807]},
    {"property": "ratio", "operator": "is greater than", "values": [1e100]},
    {"property": "credit", "operator": "is less than", "values": [1000.00]},
    {"property": "properties.amount", "operator": "is greater than", "values": [100]},
    {"property": "properties.status", "operator": "is one of", "values": [100, "100", true, null]},
    {"property": "properties.note", "operator": "is", "values": [""]},
    {"property": "properties.active", "operator": "is true"}
  ]
}
```

`age` rifiuta `["18"]`; `account` accetta `[9223372036854775807]` e rifiuta `["9223372036854775807"]`. Su JSON, `["100"]` esprime confronto testuale e `[100]` confronto numerico. Boolean e Null sono operandi validi **soltanto** per `is one of` e `is not one of`; Array e Object non sono operandi pubblici ammessi. Gli operatori senza operandi conservano i controlli attuali su booleani, null, esistenza e vuoto.

### 3.2 Matrice tipo JSON → valore interno

| Proprietà o elemento di array | Kind dell'operando API | Valore in `state.Where` | Validazione della soglia |
|---|---|---|---|
| `string` | String | `string` | Controlli testuali correnti; lista ammessa se presente |
| `int(8/16/24/32/64)` signed/unsigned | Number | `int` / `uint` | Sintassi intera; range signed/unsigned 64 bit delle soglie correnti, senza limiti più stretti della bit size |
| `float` | Number | `float64` | Parsing e validazione correnti alla bit size della proprietà; **nessun limite 76/37** |
| `decimal(P,S)` | Number | `decimal.Decimal` | Solo limiti globali 76/37; non quelli specifici P,S |
| `datetime` | String ISO8601 | `time.Time` | Parser ISO esistente e controlli correnti; nessuna nuova normalizzazione SQL |
| `date` | String `YYYY-MM-DD` | `time.Time` | Regole correnti, data UTC |
| `time` | String ora con frazione opzionale | `time.Time` | Parser corrente, ancoraggio al 1970-01-01 UTC |
| `year` | Number | `int` | Sintassi intera, da 1 a 9999 |
| `uuid`, `ip` | String | `string` normalizzata | `NormalizeUUID` / `NormalizeIP` |
| `json`, anche percorso interno JSON | String / Number; anche Boolean / Null per membership | `json.Value` del kind originale | Matrice §5; Number entro 76/37 |
| `array(T)`, soltanto dove oggi ammesso | Come T per l'operatore applicabile | Operando scalare come T | Non introdurre operandi Array; `array(json)` con `contains` accetta solo String/Number |
| `boolean`, `object`, `map` | Nessun operando aggiuntivo | Nessuno | Restano i soli operatori oggi applicabili |

La matrice conserva l'applicabilità attuale per le proprietà ordinarie, inclusi i divieti su enum stringa e membership UUID/IP. I vincoli dei **dati** non si applicano integralmente alle **soglie**: imporrebbero P,S del decimal e il range del singolo `int(n)`, oggi non imposti ai filtri. Per int64 il decoder dei filtri accetta Number, distinguendosi intenzionalmente dall'attuale `types.Decode`; gli altri vincoli non cambiano. [Validazione attuale](../../core/filters.go#L788), [parsing soglie](../../core/filters.go#L908), [types.Decode](../../tools/types/value.go#L463).

Conservare la collocazione attuale del controllo enum: l'eccezione ai limiti testuali riguarda una proprietà `string` con valori ammessi; `array(string)` passa dai controlli generici sull'operando elemento. L'ammissione di `""` riguarda JSON, non cambia questi controlli ordinari. [Ramo enum e discesa nell'elemento](../../core/filters.go#L897).

### 3.3 Numeri: valore, precisione e forma

La validazione è **specifica del tipo**, non un passaggio decimal comune a tutti i Number:

- **Int e year:** richiedere kind Number e rappresentazioni sintatticamente intere, anche per int64. `100` e `-100` sono validi quando compatibili con il tipo; rifiutare `100.0`, `1e2` e qualsiasi String numerica. Verificare la sintassi prima di qualsiasi normalizzazione che cancelli frazione/esponente; quindi usare parsing base 10 esatto, come `strconv.ParseInt` / `ParseUint` a 64 bit, senza float64. Per le soglie int conservare il range signed/unsigned a 64 bit, senza applicare `t.IntRange()` / `UnsignedRange()` della bit size della proprietà; per year mantenere 1–9999. Riutilizzare le regole lessicali pertinenti dei parser attuali senza adottare la scelta String/Number di `types.Decode` per int64, né `Decimal.Int64` / `Uint64` per ampliare la grammatica. [Decoder dati](../../tools/types/value.go#L463), [parseInt/parseUnsigned](../../core/filters.go#L491), [parseYear](../../core/filters.go#L546).
- **Float:** mantenere `parseFloat` / parsing alla bit size, finitezza e regole correnti. `1e100` resta valido per float64 e viene rifiutato se fuori range per float32. Non introdurre il controllo `types.IsDecimal` né consentire String speciali `NaN`/Infinity tramite il percorso dati di `types.Decode`. [parseFloat](../../core/filters.go#L366), [validazione](../../core/filters.go#L927), [differenza dal decoder dati](../../tools/types/value.go#L492).
- **Decimal e Number su JSON:** verificare JSON valido e kind Number, poi `decimal.Parse(..., 0, 0)` e `types.IsDecimal`. Conservare tutta la precisione, senza passaggi attraverso float64. Per la proprietà JSON il valore interno resta `json.Value`, non il decimal temporaneo. [parseFilterDecimal](../../core/filters.go#L339), [IsDecimal](../../tools/types/value.go#L82).

**P. Normalizzazione del solo Number decimal/JSON:** usare il decimal già validato e `Decimal.MarshalJSON` per produrre un token numerico equivalente. `100`, `100.0` e `1e2` possono avere una grafia canonica comune; lo zero diventa `0`. Kind e valore restano invariati. Non normalizzare prima della verifica lessicale degli interi, né far transitare i float da questo percorso. String, Boolean e Null conservano il kind originale. [MarshalJSON](../../tools/decimal/decimal.go#L210).

Il limite 76/37 riguarda il **valore rappresentabile**, non la lunghezza della grafia. `PrecisionScale` ignora gli zeri frazionari finali; `0.005` richiede precisione 3 e scala 3, zero 1/0. `1e75` e `1e-37` sono ammessi, `1e76` e `1e-38` no. Per `decimal(5,2)` la soglia `1000.00` resta valida: non chiamare `types.Decode` con P,S della proprietà. Per serializzare decimal usare `MarshalJSON`, non `String()`, che può conservare forme non JSON come `+5` o `.5`. [PrecisionScale](../../tools/decimal/decimal.go#L233), [test limiti](../../tools/types/value_test.go#L345).

**C.** `decimal.Parse("1e100", 0, 0)` può restituire un decimal fuori 76/37; il controllo successivo lo rifiuta come soglia decimal/JSON, non come soglia float64. Il decoder dei dati decimal applica invece P,S e `DecimalRange` della proprietà: non è il decoder corretto delle soglie. [Decode decimal](../../tools/types/value.go#L518), [vincoli parser](../../tools/decimal/decimal.go#L627).

**C. Limite del parser nel checkout:** `Parse` accetta anche forme non JSON e non sostituisce `json.Valid`. Inoltre `0e1000000000000000000`, JSON valido uguale a zero, incontra il controllo di range prima del riconoscimento della mantissa zero. La correzione autonoma `9c45ac229` già preparata nella sessione non è nella baseline: verificarne la disponibilità nella futura implementazione, senza duplicare l'eccezione nei filtri. Anche con tale correzione restano necessari i controlli 76/37 per decimal/JSON. [Forme accettate](../../tools/decimal/decimal.go#L466), [esponente e zero](../../tools/decimal/decimal.go#L581).

## 4. Deserializzazione e validazione schema-aware

### 4.1 `cmd`: sola struttura, indipendenza dall'ordine

Cambiare la variabile `values` di `unmarshalFilterRule` in `[]json.Value`, mantenendo il decoder strutturale esistente. Ogni elemento deve essere posseduto dal filtro: `json.Value.UnmarshalJSON` copia i byte, mentre `Decoder.ReadValue` ha durata limitata. Preferire `Decode(&values)` e verificarne il comportamento con il nuovo tipo. [ReadValue](../../tools/json/decoder.go#L75), [UnmarshalJSON](../../tools/json/value.go#L397), [decode filtro](../../core/filters.go#L645).

Mantenere campi sconosciuti/duplicati, radice gruppo, limiti 4/100, esclusione `rules`/`property` e divieto di `values` nei gruppi. Nessuna risoluzione dello schema o conversione numerica in `cmd`: in `PipelineToSet`, `filter` prima o dopo `inSchema` deve dare lo stesso risultato. [CreatePipeline](../../cmd/connection.go#L84), [Update](../../cmd/pipeline.go#L127), [listener](../../cmd/workspaces.go#L158), [eventi](../../cmd/workspaces.go#L221), [profili](../../cmd/workspaces.go#L878).

| Forma di `values` | Significato strutturale | Regola |
|---|---|---|
| Omesso oppure `null` | Slice nil, nessun operando | Ammesso per gli operatori senza valori, come oggi |
| `[]` | Slice non nil vuota | Rifiutato dagli unari secondo la regola attuale |
| `[null]` | Un operando JSON Null esplicito | Valido solo per membership JSON; rifiutato dagli unari |
| `[]json.Value{nil}` costruito in Go | Operando raw non valido | Rifiutare; non interpretarlo come JSON Null |

Il Null esplicito è `json.Value("null")`, non la slice raw nil. Il marshal emette direttamente i documenti JSON attraverso `json.Buffer.Encode`, senza quoting ulteriore e senza validazione schema-aware. Verificare anche il marshal/unmarshal standalone di `FilterCondition` e l'unmarshal ripetuto nella stessa destinazione: `json.Value.UnmarshalJSON` rifiuta receiver già non vuoti. Se serve, usare una destinazione temporanea locale, senza cambiare globalmente `tools/json.Value`. [Marshal](../../core/filters.go#L422), [test marshal](../../core/filters_test.go#L71), [test unmarshal](../../core/filters_test.go#L254).

### 4.2 `core`: preparazione in una visita

**P.** Sostituire la coppia interna `validateFilter` + `convertFilterToWhere` con:

```go
func prepareFilter(filter *Filter, schema types.Type) (*state.Where, []string, error)
```

La visita valida anche gli alberi costruiti in Go, risolve proprietà/tipo una volta, verifica cardinalità e compatibilità dell'operatore, decodifica ciascun operando una volta e costruisce `state.Where`. Restituisce le stesse proprietà base uniche/ordinate oggi raccolte, comprese le basi JSON; copia i byte raw e non modifica né memorizza cache nel filtro pubblico mutabile. Per `nil` il risultato è `nil, nil, nil`; per un filtro presente lo schema deve essere valido.

**Decoder specifico degli operandi, decisione definitiva.** Prendere come riferimento `types.Decode`, riusandone i parser opportuni, ma senza chiamarlo indiscriminatamente sul tipo della proprietà. Evolvere il precedente `state.unmarshalConditionValue` in un convertitore di **un singolo operando**, utilizzabile da preparazione core e lettura dello stato, per esempio:

```go
// DecodeFilterOperand validates and converts a filter operand for typ and operator.
func DecodeFilterOperand(value json.Value, typ types.Type, operator WhereOperator) (any, error)
```

Collocazione proposta: `core/internal/state/where.go`, già importabile da `core`, senza dipendenze da `core.Filter`. Il contesto operatore distingue Boolean/Null validi in membership da quelli vietati altrove. Il decoder verifica raw JSON, kind, sintassi e rappresentabilità secondo §3; riusa parsing interi/float, decimal senza P,S, ISO/date/time e normalizzatori UUID/IP. Per gli interi richiede sempre Number e applica i controlli lessicali prima della conversione `strconv`, senza distinguere il kind in base alla bit size né delegare al decoder generale attuale; per float conserva il contratto dei filtri. Enum, limiti testuali, schema/percorso e cardinalità restano nel preparatore: non trasformare il decoder in un validatore completo della richiesta. L'adeguamento futuro di `types.Decode` resta indipendente. [Precedente](../../core/internal/state/where.go#L339), [types.Decode](../../tools/types/value.go#L463).

Il decoder datetime conserva l'offset ricevuto nel percorso API, come fa oggi la conversione del filtro. Il caricamento persistito può mantenere esplicitamente la propria normalizzazione UTC già esistente (§4.3 e §6), senza imporla a tutti i chiamanti del decoder condiviso.

Gli errori del helper non implementano `ResponseWriterTo`; il metodo API li traduce nella classificazione attuale. Conservare `types.PathNotExistError`: `Workspace.Profiles` lo distingue come `Unprocessable(PropertyNotExist)`, mentre eventi e utenti applicativi producono `BadRequest`. [Profiles](../../core/workspace.go#L1279), [Events](../../core/workspace.go#L941), [ApplicationUsers](../../core/connection.go#L240).

Per creazione/aggiornamento pipeline rendere disponibile il risultato dopo `validatePipelineToSet`: aggiungere a `validationState` un risultato interno `filter *state.Where`, passare `*validationState` al validatore e assegnare il risultato solo al successo. Azzerare l'output se il contenitore viene riusato. Creazione e aggiornamento usano quel risultato in `MarshalWhere`, evitando la seconda preparazione. [validationState](../../core/pipeline_validate.go#L20), [filtro](../../core/pipeline_validate.go#L150), [CreatePipeline](../../core/connection.go#L400), [Update](../../core/pipeline.go#L778).

### 4.3 Validazione di valori raw e confini

`json.Value` presume JSON valido: `Kind()` su raw vuoto può andare in panic e `String()` restituisce testo anche per kind non String. Verificare non-vuoto e `json.Valid` prima degli accessi; rifiutare raw nil, documenti multipli e token invalidi. Il documento valido `null` è invece ammesso nel contesto membership JSON. [Kind](../../tools/json/value.go#L233), [String](../../tools/json/value.go#L353), [MarshalJSON](../../tools/json/value.go#L284).

Conservare UTF-8, divieto di NUL, 80 rune ed eccezione enum; per JSON ammettere il contenuto String vuoto secondo A4. Non applicare il limite testuale ai byte dei Number: usare limiti numerici specifici del tipo e controlli di risorsa, senza costruire potenze enormi. Il body è limitato a 500 KiB; quel controllo non stabilisce un limite dedicato delle query string filtro. [ValidateStringField](../../core/internal/util/util.go#L79), [body](../../cmd/apis_server.go#L33), [reader](../../cmd/apis_server.go#L199).

**Datetime, perimetro definitivo:** mantenere parser e semantica SQL attuali, compresa la gestione dell'offset nel `time.Time` preparato; non normalizzare ora tutto in UTC né cambiare i literal dei renderer. La conversione inversa attuale omette l'offset: il nuovo serializer API deve includerlo per conservare l'istante nel round-trip. La persistenza attuale scrive/legge UTC: preservare questa politica, senza usarla per cambiare anche i percorsi diretti API→SQL. Date/time mantengono formati e ancoraggio correnti. Eventuali correzioni ulteriori, incluso il parser permissivo di time e gli intervalli invertiti, restano attività separate. [Conversione](../../core/filters.go#L233), [inversa](../../core/filters.go#L269), [persistenza UTC](../../core/internal/state/where.go#L321), [ParseTime](../../core/internal/util/util.go#L16), [ISO vendor](../../vendor/github.com/relvacode/iso8601/iso8601.go#L352).

## 5. Semantica degli operatori JSON

### 5.1 Regola fondamentale e negazioni

Il kind dell'**operando** determina il dominio del confronto. Number usa `Decimal.Equal` / `Cmp`, String il contenuto decodificato e l'ordinamento corrente; niente confronto numerico sui byte raw. Boolean e Null partecipano soltanto alla membership, rispettando il proprio kind.

| Dato JSON | `is greater than [100]` | `is greater than ["100"]` |
|---|---:|---:|
| `150` | true | false |
| `9` | false | false |
| `"150"` | false | true |
| `"9"` | false | true |
| `null`, Boolean, oggetto, array, assente | false | false |

Per `is not`, `is not between`, `does not contain` e `is not one of` preservare il complemento booleano del predicato positivo. `is false` resta un controllo del booleano falso, non il complemento di `is true`. La membership con Null è l'eccezione esplicita alla regola «null non soddisfa confronti positivi»:

| Dato rispetto a `values: [100, "100", true, null]` | `is one of` | `is not one of` |
|---|---:|---:|
| Proprietà assente / estrazione SQL NULL | false | true |
| JSON Null presente | true | false |
| Go nil con proprietà presente, equivalente al null del contratto corrente | true | false |
| Number `100`, `100.0`, `1e2` | true | false |
| String `"100"` | true | false |
| Boolean `true` | true | false |
| Boolean `false` | false | true |
| String `"true"` oppure `"null"` | false | true |
| Array oppure Object | false | true |

La lista `[null]` seleziona soltanto JSON Null **presente**; `[false]` seleziona soltanto Boolean false. Una lista priva di Null non seleziona JSON Null. Duplicati e ordine degli operandi mantengono il trattamento attuale; nessuna uguaglianza strutturale di Array/Object viene esposta nel filtro pubblico.

| Stato della proprietà JSON | Confronto S/N positivo | Sua negazione | `is null` | `exists` | `is empty` |
|---|---:|---:|---:|---:|---:|
| Assente / estrazione SQL NULL | false | true | false | false | true |
| JSON Null presente oppure Go nil presente | false | true | true | true | true |
| Presente, kind incompatibile | false | true | false | true | Dipende dal contenuto |
| Presente, kind confrontabile | Risultato | Complemento | false | true | Dipende dal contenuto |

`is not null`, `does not exist` e `is not empty` restano complementi; 0 e false non sono vuoti. Il lettore Go restituisce presenza separata dal valore. Il decoder dei dati JSON rappresenta normalmente null con `json.Value("null")`; il nil presente descrive anche le mappe Go già costruite accettate dal valutatore corrente. SQL NULL non deve imitare JSON Null. I rami delle proprietà ordinarie non ereditano automaticamente questa tabella. [Lettura](../../core/internal/filters/applies.go#L496), [null/empty](../../core/internal/filters/applies.go#L470), [decode null](../../tools/types/value.go#L326), [unari PG](../../warehouses/postgresql/renderexpr.go#L490), [unari Snowflake](../../warehouses/snowflake/renderexpr.go#L490).

### 5.2 Catalogo di tutti i 26 operatori

**Decisioni definitive.** `S` = String, `N` = Number, `B` = Boolean, `Z` = Null. La cardinalità non cambia. Le righe con operandi si applicano alle proprietà JSON secondo questa matrice; l'applicabilità dei tipi ordinari resta quella corrente.

| # | Operatore | Operandi JSON | Regola |
|---:|---|---|---|
| 1 | `is` | 1 S oppure N | Uguaglianza entro lo stesso kind |
| 2 | `is not` | 1 S oppure N | Complemento di `is` |
| 3 | `is less than` | 1 S oppure N | Ordinamento entro lo stesso kind |
| 4 | `is less than or equal to` | 1 S oppure N | Come sopra, inclusivo |
| 5 | `is greater than` | 1 S oppure N | Ordinamento entro lo stesso kind |
| 6 | `is greater than or equal to` | 1 S oppure N | Come sopra, inclusivo |
| 7 | `is between` | 2 S oppure 2 N | Estremi inclusi, stesso kind obbligatorio |
| 8 | `is not between` | Come `is between` | Complemento |
| 9 | `contains` | 1 S oppure N | S: sottostringa o elemento String immediato; N: elemento Number immediato |
| 10 | `does not contain` | Come `contains` | Complemento |
| 11 | `is one of` | ≥1 S/N/B/Z, anche misti | OR di uguaglianze per kind, Null presente distinto da assenza |
| 12 | `is not one of` | Come `is one of` | Complemento |
| 13 | `starts with` | 1 S | Solo dato String |
| 14 | `ends with` | 1 S | Solo dato String |
| 15 | `is before` | Non applicabile a JSON | Resta riservato ai tipi temporali |
| 16 | `is on or before` | Non applicabile a JSON | Invariato |
| 17 | `is after` | Non applicabile a JSON | Invariato |
| 18 | `is on or after` | Non applicabile a JSON | Invariato |
| 19 | `is true` | Nessuno | Boolean true presente |
| 20 | `is false` | Nessuno | Boolean false presente |
| 21 | `is empty` | Nessuno | Semantica corrente di vuoto/assenza |
| 22 | `is not empty` | Nessuno | Complemento |
| 23 | `is null` | Nessuno | JSON Null presente |
| 24 | `is not null` | Nessuno | Complemento, include assenza |
| 25 | `exists` | Nessuno | Percorso presente, anche JSON Null |
| 26 | `does not exist` | Nessuno | Complemento |

Gli otto unari richiedono ancora `Values == nil`; `[null]` non è un modo alternativo di invocarli. Per `exists` restano proprietà `ReadOptional` o percorsi interni JSON, senza estensione automatica a colonne JSON base obbligatorie. [Catalogo](../../core/filters.go#L135), [validazione](../../core/filters.go#L788), [cardinalità](../../core/filters.go#L875).

### 5.3 Decisioni A1–A6 e cambiamenti intenzionali

| ID | Comportamento attuale | Regola approvata e conseguenza |
|---|---|---|
| A1 — `between` | Ogni estremo testuale può avere ramo String e Number | Entrambi String oppure entrambi Number; coppie miste rifiutate. Conservare ordine e inclusività, senza riordinamento né nuovo controllo primo ≤ secondo |
| A2 — membership | Testi duali; Boolean/Null non confrontati come tali | Liste miste S/N/B/Z; nessuna coercizione. JSON Null presente può fare match, assenza mai |
| A3 — `contains` | Sottostringa e appartenenza con operando duale | String cerca sottostringa o elemento String immediato; Number soltanto elemento Number immediato in array. Nessuna ricerca ricorsiva |
| A4 — String vuota | Normalmente rifiutata da validator e controllo | `""` esplicito ammesso per JSON; controllo editoriale vuoto resta incompleto. Non allentare le regole ordinarie |
| A5 — compatibilità | Tutti gli operandi pubblici sono testi | Matrice §5.2; Boolean/Null aggiunti esclusivamente alla membership, prefisso/suffisso solo String |
| A6 — interi/year | Soglie con sintassi intera | Conservare la restrizione sintattica su tutti gli interi Number; `100.0`, `1e2` e String numeriche rifiutati, anche per int64 |

Con A4, `contains ""`, `starts with ""` e `ends with ""` sono veri per qualsiasi dato String, come già previsto dalle funzioni sottostanti; su array `contains ""` cerca soltanto un elemento String vuoto. È una conseguenza della decisione approvata, non una questione ancora aperta. [Contains](../../core/internal/filters/applies.go#L277), [prefisso/suffisso](../../core/internal/filters/applies.go#L370), [fixture interne](../../test/warehouseexpr/json.go#L83).

## 6. Rappresentazione interna e persistenza

### 6.1 `state.Where` e conversione inversa

Conservare `Values []any` e i tipi Go della matrice §3.2. Tutti gli interi, compresi int64, diventano `int` / `uint` dal Number API; non attraversano float64. Per JSON sostituire il duale con `json.Value`, copiato e trattato come immutabile:

```go
membership := &state.WhereCondition{
    Property: []string{"properties", "status"},
    Operator: state.OpIsOneOf,
    Values: []any{
        json.Value("100"),
        json.Value("\"100\""),
        json.Value("true"),
        json.Value("null"),
    },
}
```

Eliminare `state.JSONConditionValue` quando gli usi sono convertiti, senza fallback. `convertWhereToFilter` codifica tutti gli int/year come Number, float come Number con le regole correnti, decimal con `MarshalJSON`, tempi/UUID/IP come String e JSON con copia diretta. Per gli interi usare la formattazione decimale esatta di `int` / `uint`, senza consultare la bit size: l'attuale `types.Marshal` quota int64 e non va usato per questo ramo finché il contratto generale non sarà aggiornato separatamente. Gli altri formatter possono essere riusati dove coincidono con le soglie, senza delegare loro indiscriminatamente la validazione. Per datetime includere l'offset nella String API per conservare l'istante; nessuna nuova normalizzazione UTC nella conversione inversa. [Inversa attuale](../../core/filters.go#L269), [JSONConditionValue](../../core/internal/state/where.go#L224), [marshal tipi](../../tools/types/value.go#L635).

### 6.2 Formato persistito

L'albero mantiene operatori interni e percorsi a segmenti. Gli operandi JSON diventano valori diretti; tutti gli interi, inclusi signed/unsigned a 64 bit, sono Number:

```json
{
  "operator": "And",
  "rules": [
    {"property": ["account"], "operator": "Is", "values": [9223372036854775807]},
    {"property": ["properties", "status"], "operator": "IsOneOf", "values": [100, "100", true, null]}
  ]
}
```

Aggiornare `marshalWhereValue` aggiungendo `json.Value` diretto; conservare i casi `int`/`uint` che già emettono Number mediante `strconv.FormatInt` / `FormatUint`. Non serve conoscere la bit size per scegliere fra String e Number, né aggiungere parametri schema per tale distinzione. `MarshalWhere` può mantenere il suo parametro schema per le responsabilità già esistenti, come la formattazione degli altri tipi: non occorre un refactoring delle firme. `whereConditionJSON.Values` è già `[]json.Value`; il loader usa `DecodeFilterOperand` con tipo **e operatore**, mantenendo `[null]` come un operando e ricostruendo `int`/`uint` da Number senza float64. Lo schema resta necessario alla decodifica per distinguere tipi e signedness. Non interpretare una String JSON come secondo documento JSON. [Serializer](../../core/internal/state/where.go#L301), [decoder](../../core/internal/state/where.go#L339), [struttura raw](../../core/internal/state/where.go#L230), [load](../../core/internal/state/load.go#L617).

Normalizzare esattamente i Number decimal/JSON durante la preparazione/decodifica; float e interi seguono i propri percorsi. Gli interi sono serializzati in notazione intera senza frazione/esponente, dopo la verifica sintattica in ingresso. Per datetime mantenere la scrittura UTC e la lettura UTC già implementate nella persistenza; ciò non autorizza una variazione dei confronti SQL diretti. Nessun campo versione, doppia lettura o conversione del vecchio formato viene progettato: resta soltanto l'avvertenza di §11. [Persistenza datetime](../../core/internal/state/where.go#L321), [notifiche](../../core/internal/state/keep.go#L629).

### 6.3 Uguaglianza

`Where.Equal` resta strutturale e ordinato; `WhereCondition.Equal` confronta i valori JSON semanticamente e per kind: String decodificate, Number tramite `Decimal.Equal`, Boolean uguali soltanto allo stesso booleano, Null uguale soltanto a Null. `100` = `100.0` = `1e2`, ma Number 100 ≠ String `"100"`; false ≠ Null. Slice raw nil non è un operando Null valido. `reflect.DeepEqual` dei byte non basta; nessuna equivalenza logica aggiunta fra gruppi riordinati o duplicati. [Equal](../../core/internal/state/where.go#L106), [uso nelle destinazioni](../../core/internal/collector/destinations.go#L381).

## 7. Modifiche al valutatore Go

`filters.Applies` mantiene firma, ricorsione e short-circuit. Il filtro è preparato; i record continuano a fornire tipi Go ordinari e `json.Value` per JSON. [Applies](../../core/internal/filters/applies.go#L17).

Nei rami JSON di `opIs`, ordinamenti, `opIsBetween`, `opContains`, `opIsIn`, prefissi/suffissi:

1. Leggere il kind dell'operando `json.Value` prima del contenuto; Number confrontato solo con Number tramite decimal esatto, String solo con String. Nessun confronto raw o conversione String→Number.
2. Applicare A1 e A3 agli intervalli e agli elementi immediati degli array. Boolean/Null non diventano validi per `is`, ordinamenti o `contains`.
3. Per membership aggiungere Boolean e Null con tipo rigoroso. Controllare il kind Boolean prima di leggere `Bool()`: un getter che restituisce false su altro kind non deve far coincidere false con Null o Number.
4. Conservare la presenza nella membership. Oggi `appliesCondition` la legge ma passa a `opIsIn` soltanto il valore; occorre passarla, per esempio `opIsIn(v, values, exists)`, oppure applicare un guard equivalente nel dispatcher. Missing dà sempre false al positivo; un nil **presente** può corrispondere a Null, come nel contratto corrente di `is null`.
5. Negare il risultato positivo completo, non il singolo confronto senza i guard: la proprietà assente soddisfa `is not one of` anche quando la lista contiene Null.

I rami di membership ordinaria rimangono invariati. Non usare `==` su `json.Value` nelle interfacce: sono slice non confrontabili. Riutilizzare un confronto scalare JSON coerente con §5, senza riaprire uguaglianza strutturale per Array/Object nel filtro pubblico. [Dispatcher e presenza](../../core/internal/filters/applies.go#L44), [is](../../core/internal/filters/applies.go#L106), [between](../../core/internal/filters/applies.go#L242), [contains](../../core/internal/filters/applies.go#L277), [membership](../../core/internal/filters/applies.go#L307), [null](../../core/internal/filters/applies.go#L470).

Il limite 76/37 riguarda la soglia decimal/JSON validata, non ogni Number letto dal record né una soglia float. Il parsing del dato resta distinto da quello della costante; conservare i confronti senza float64 e coprire dati con esponenti ampi già presenti nelle fixture. Gli unari e la gestione temporale non richiedono variazioni funzionali. [Fixture datastore](../../core/internal/datastore/query_json_test.go#L95).

## 8. Modifiche ai compilatori PostgreSQL e Snowflake

### 8.1 Datastore e AST: separare l'intenzione del confronto

`BaseExpr.Values []any` trasporta `json.Value`; `MultiExpr`, operatori, `RowQuery`, join, paginazione e conteggi non cambiano. Rimuovere l'adattatore `state.JSONConditionValue → warehouses.JSONConditionValue` da `datastore.convertWhere`, conservando mapping colonne/chiavi, gruppi e riferimenti `warehouses.Column` già validi. Eliminare il duale warehouse dopo l'adeguamento dei consumatori. [BaseExpr](../../warehouses/warehouses.go#L824), [duale](../../warehouses/warehouses.go#L854), [convertWhere](../../core/internal/datastore/query.go#L99).

**C. Collisione esistente, anche sugli scalari.** Su `is` / `is not`, entrambi i renderer interpretano già un `json.Value` come uguaglianza JSON strutturale. Non basta fare dispatch per kind e riservare il ramo strutturale a Object/Array: un test esistente usa proprio `json.Value("1e76")` per un confronto strutturale fuori dai limiti delle soglie decimal. Anche String, Number, Boolean e Null strutturali devono conservare il loro percorso. [PG](../../warehouses/postgresql/renderexpr.go#L379), [Snowflake](../../warehouses/snowflake/renderexpr.go#L362), [test PG](../../warehouses/postgresql/renderexpr_test.go#L337), [test Snowflake](../../warehouses/snowflake/renderexpr_test.go#L323).

**P.** Aggiungere un discriminante dell'intenzione a `BaseExpr`, senza cambiare il tipo dei valori né la firma di `NewBaseExpr`. Esempio di interfaccia proposta:

```go
// JSONComparisonMode identifies how JSON operands are compared.
type JSONComparisonMode uint8

const (
    JSONStructuralComparison JSONComparisonMode = iota
    JSONTypedComparison
)

// In BaseExpr, alongside the existing fields:
// JSONComparison JSONComparisonMode
```

Lo zero conserva il confronto strutturale già usato da `NewBaseExpr(..., OpIs/OpIsNot, json.Value(...))`. `datastore.convertWhere` imposta esplicitamente `JSONTypedComparison` per i predicati dei filtri JSON, compreso `contains` su `array(json)`; i nuovi costruttori/fixture tipizzati fanno lo stesso. Gli unari non dipendono dal discriminante. Con operandi raw, gli altri confronti JSON richiedono esplicitamente la modalità tipizzata. Questo percorso valida kind/cardinalità e la matrice §5; quello strutturale mantiene la capacità interna di uguaglianza, senza renderla pubblica nell'API dei filtri. Il discriminante è metadato dell'AST SQL, non un formato/versione di filtro né un wrapper sostitutivo di `JSONConditionValue`.

Non validare indiscriminatamente tutti i raw JSON Number come decimal 76/37 **prima** di scegliere il percorso: romperebbe il test strutturale `1e76`. Il limite vale per Number di confronti JSON tipizzati e per gli operandi `decimal.Decimal` ordinari, come oggi; non per float né per tutti i JSON strutturali. Validare comunque raw JSON e modalità AST prima dell'uso. Mantenere le due implementazioni uniformi, localizzando solo le differenze di dialetto. [Validazione attuale dei decimal](../../warehouses/postgresql/renderexpr.go#L66).

### 8.2 Confronto sistematico fra SQL attuale e previsto

Le forme di questa tabella sono **schematiche**, ricostruite dai renderer attuali; non sono snapshot del rendering futuro. `v` rappresenta l'estrazione JSONB in PG, per esempio `("properties" -> 'value')`, e VARIANT in Snowflake, per esempio `GET("PROPERTIES", 'value')`. L'estrazione effettiva, il quoting e le collation correnti vengono riusati. I rendering già confermati da test sono distinti in §8.6.

| Caso | PostgreSQL attuale → previsto | Snowflake attuale → previsto |
|---|---|---|
| `is` e ordinamenti JSON, testo API `"100"` | CASE String + Number quando il testo è numerico → un solo dominio scelto dal kind dell'operando | CASE VARCHAR + DECIMAL/DOUBLE → un solo dominio; Number mantiene `renderJSONNumberComparison` |
| Operando Number `100` | Nuovo formato API; ora la soglia numerica è il ramo del duale → solo guard Number e cast numeric esatto | Nuovo formato API → solo guard DECIMAL/DOUBLE e confronto DECFLOAT esistente |
| Operando String `"100"` | Ramo String più opportunistico Number → solo `#>> '{}'`, `COLLATE "C"` | Ramo VARCHAR più opportunistico numerico → solo `AS_VARCHAR`, collation `utf8` |
| `between` | Due confronti duali → due confronti dello stesso kind, bounds inclusivi | Stessa restrizione; mantenere l'algoritmo numerico per ciascuna soglia Number |
| Membership S/N | Tutti i testi in IN String, più quelli parsabili in IN Number → partizioni per kind effettivo | Stessa separazione; nessuna duplicazione automatica String→Number |
| Membership Boolean/Null | Non disponibile come confronto pubblico tipizzato → nuovi rami Boolean e JSON Null | Non disponibile → ramo `AS_BOOLEAN` e `IS_NULL_VALUE` |
| Negazioni JSON | `NOT COALESCE(positivo, FALSE)` → stessa forma su positivo tipizzato completo | Identico contratto, inclusa assenza e kind incompatibile |
| `contains` | Sottostringa più elementi String/Number del duale → S: sottostringa/elementi S; N: solo elementi N immediati | Stessa regola tramite ricerca String e `FILTER` sugli array |
| Prefisso/suffisso | Testo del duale → solo String esplicita | Solo String esplicita, funzioni testuali correnti |
| Unari JSON | Null, booleani, esistenza e vuoto → invariati | Invariati, distinguendo SQL NULL da JSON Null |
| Uguaglianza strutturale interna | `v = literal::jsonb` → stessa espressione con modalità strutturale | `v = PARSE_JSON(literal)` → stessa espressione con modalità strutturale |
| Predicati ordinari | Stessi tipi Go → stessi operatori/literal, compresi comportamento SQL NULL e datetime | Stessi tipi Go → stessi operatori/literal e limiti nativi del dialetto |

Riferimenti del confronto: [PG confronto](../../warehouses/postgresql/renderexpr.go#L256), [membership](../../warehouses/postgresql/renderexpr.go#L434), [Snowflake confronto](../../warehouses/snowflake/renderexpr.go#L243), [membership](../../warehouses/snowflake/renderexpr.go#L416), [confronto numerico](../../warehouses/snowflake/renderexpr.go#L510).

### 8.3 Esempi String/Number: prima e dopo

**SQL schematico attuale PG**, ricostruito da `renderJSONComparison`, per `is greater than` con la vecchia soglia testuale `"100"`:

```sql
COALESCE(CASE
  WHEN jsonb_typeof(v) = 'string' THEN (v #>> '{}') COLLATE "C" > '100'
  WHEN jsonb_typeof(v) = 'number' THEN (v)::numeric > 100
  ELSE FALSE END, FALSE)
```

**SQL schematico previsto PG**, rispettivamente per Number `100` e String `"100"`:

```sql
-- values: [100]
COALESCE(CASE WHEN jsonb_typeof(v) = 'number'
  THEN (v)::numeric > 100 ELSE FALSE END, FALSE)
-- values: ["100"]
COALESCE(CASE WHEN jsonb_typeof(v) = 'string'
  THEN (v #>> '{}') COLLATE "C" > '100' ELSE FALSE END, FALSE)
```

Il cast numeric resta nel ramo Number del CASE; non affidarne la sicurezza all'ordine di due condizioni AND. Non confrontare rappresentazioni testuali dei Number. La stessa separazione vale per `=`, `<`, `<=` e `>=`.

**SQL schematico attuale Snowflake**, ricostruito dallo stesso caso:

```sql
COALESCE(CASE
  WHEN IS_VARCHAR(v) THEN COLLATE(AS_VARCHAR(v), 'utf8') > '100'
  WHEN IS_DECIMAL(v) OR IS_DOUBLE(v)
    THEN TRY_TO_DECFLOAT(TO_JSON(v)) > TO_DECFLOAT('1e2')
  ELSE FALSE END, FALSE)
```

**SQL schematico previsto Snowflake**:

```sql
-- values: [100]
COALESCE(CASE WHEN IS_DECIMAL(v) OR IS_DOUBLE(v)
  THEN TRY_TO_DECFLOAT(TO_JSON(v)) > TO_DECFLOAT('1e2')
  ELSE FALSE END, FALSE)
-- values: ["100"]
COALESCE(CASE WHEN IS_VARCHAR(v)
  THEN COLLATE(AS_VARCHAR(v), 'utf8') > '100' ELSE FALSE END, FALSE)
```

Riutilizzare `renderJSONNumberComparison` per **tutti** i nuovi confronti Number tipizzati, senza sostituirlo con `PARSE_JSON` della soglia o uguaglianza di testi. Il renderer tratta numeri JSON memorizzati fino a 38 cifre significative; per soglie con coefficiente più preciso l'uguaglianza numerica non trova un dato uguale, mentre l'ordinamento tronca la soglia a 38 cifre correggendo l'inclusività secondo il segno del resto. I limiti dell'operando restano 76/37. [Algoritmo e commento](../../warehouses/snowflake/renderexpr.go#L510), [jsonNumberParts](../../warehouses/snowflake/renderexpr.go#L23), [test numerici](../../warehouses/snowflake/jsonexpr_test.go#L22).

I test di storage mostrano trasformazioni della rappresentazione; i differenziali rileggono il dato memorizzato e valutano Go su quello. Mantenere tale criterio: l'esattezza della soglia non ricostruisce cifre perse nello storage. Nessun cambio di storage è richiesto. [Test storage](../../warehouses/snowflake/jsonexpr_test.go#L323), [dato riletto](../../core/internal/datastore/query_json_test.go#L216).

### 8.4 Membership mista, Null e negazioni

**Prima, schematico:** una lista valida del vecchio formato `["100", "true", "null"]` produce un IN String con i tre testi e un IN Number con 100; non seleziona Boolean true o JSON Null. Non attribuire un SQL corrente al nuovo payload `[100, "100", true, null]`: quel contratto non esiste nella baseline.

```sql
-- PostgreSQL attuale, values: ["100", "true", "null"]
COALESCE(CASE
  WHEN jsonb_typeof(v) = 'string'
    THEN (v #>> '{}') COLLATE "C" IN ('100', 'true', 'null')
  WHEN jsonb_typeof(v) = 'number' THEN (v)::numeric IN (100)
  ELSE FALSE END, FALSE)

-- Snowflake attuale, stessa lista testuale
COALESCE(CASE
  WHEN IS_VARCHAR(v) THEN COLLATE(AS_VARCHAR(v), 'utf8') IN ('100', 'true', 'null')
  WHEN IS_DECIMAL(v) OR IS_DOUBLE(v)
    THEN TRY_TO_DECFLOAT(TO_JSON(v)) IN (TO_DECFLOAT('1e2'))
  ELSE FALSE END, FALSE)
```

**Dopo, schematico PG**, per `values: [100, "100", true, null]`:

```sql
COALESCE(CASE
  WHEN jsonb_typeof(v) = 'string' THEN (v #>> '{}') COLLATE "C" IN ('100')
  WHEN jsonb_typeof(v) = 'number' THEN (v)::numeric IN (100)
  WHEN jsonb_typeof(v) = 'boolean' THEN v IN ('true'::jsonb)
  WHEN jsonb_typeof(v) = 'null' THEN TRUE
  ELSE FALSE END, FALSE)
```

**Dopo, schematico Snowflake**, per la stessa lista:

```sql
COALESCE(CASE
  WHEN IS_VARCHAR(v) THEN COLLATE(AS_VARCHAR(v), 'utf8') IN ('100')
  WHEN IS_DECIMAL(v) OR IS_DOUBLE(v)
    THEN TRY_TO_DECFLOAT(TO_JSON(v)) IN (TO_DECFLOAT('1e2'))
  WHEN AS_BOOLEAN(v) IS NOT NULL THEN AS_BOOLEAN(v) IN (TRUE)
  WHEN IS_NULL_VALUE(v) THEN TRUE
  ELSE FALSE END, FALSE)
```

Raggruppare gli operandi S/N/B e registrare la presenza di Null, senza emettere `IN ()` per gruppi vuoti. Per Boolean riusare conversioni rigorose del kind, senza cast permissivi; `AS_BOOLEAN` e `IS_NULL_VALUE` sono già usati negli unari Snowflake. I test sul warehouse devono confermare il nuovo raggruppamento anche per false e String `"false"`. Per Number mantenere le regole di rappresentabilità del renderer numerico/membership corrente. [Unari Snowflake](../../warehouses/snowflake/renderexpr.go#L386), [Null](../../warehouses/snowflake/renderexpr.go#L496).

Una membership **solo Null** può essere resa schematicamente come `COALESCE(v = 'null'::jsonb, FALSE)` in PG e `COALESCE(IS_NULL_VALUE(v), FALSE)` in Snowflake. Non usare `v IS NULL` né `v IN (NULL)`: selezionerebbero assenza o produrrebbero UNKNOWN, perdendo JSON Null presente.

Tutte le negazioni JSON rimangono `NOT COALESCE(positivo, FALSE)`: missing è false al positivo e true al negativo; JSON Null è true se e solo se la lista include Null; un kind incompatibile non soddisfa il positivo. L'involucro va applicato anche a confronti e membership con un solo ramo. Non riscrivere semplicemente in `NOT IN` lasciando propagare SQL NULL.

### 8.5 Array e operatori senza valori

Per `contains` String conservare ricerca di sottostringa e confronto con elementi String immediati; Number genera soltanto il ramo di ricerca di elementi Number immediati. Riutilizzare `jsonb_array_elements` protetto sui non-array e `unnest` per `array(json)` in PG; `FILTER`/conversioni array già esistenti in Snowflake. Confrontare gli elementi numerici semanticamente, non come token raw, senza attraversare array annidati. Boolean e Null restano vietati come operandi di `contains`. [PG](../../warehouses/postgresql/renderexpr.go#L463), [Snowflake](../../warehouses/snowflake/renderexpr.go#L462).

Preservare `is true`/`is false`, esistenza, JSON Null e vuoto con i loro controlli attuali, e le rispettive negazioni. L'aggiunta di `[null]` alla membership non cambia né `Values == nil` degli unari né la distinzione SQL NULL/JSON Null.

### 8.6 Rendering confermati da conservare e confini ordinari

Le stringhe seguenti sono **output attuali esatti presenti nei test**; il criterio previsto è conservarle identiche per gli stessi AST strutturali/ordinari:

| Caso e prova | PostgreSQL: attuale = previsto | Snowflake: attuale = previsto |
|---|---|---|
| JSON Object strutturale: [PG](../../warehouses/postgresql/renderexpr_test.go#L46), [SF](../../warehouses/snowflake/renderexpr_test.go#L40) | `COALESCE("values" = '{"foo":2,"boo":true}'::jsonb, FALSE)` | `COALESCE("VALUES" = PARSE_JSON('{"foo":2,"boo":true}'), FALSE)` |
| JSON Number strutturale fuori 76/37: [PG](../../warehouses/postgresql/renderexpr_test.go#L337), [SF](../../warehouses/snowflake/renderexpr_test.go#L323) | `COALESCE("values" = '1e76'::jsonb, FALSE)` | `COALESCE("VALUES" = PARSE_JSON('1e76'), FALSE)` |
| `between` ordinario: [PG](../../warehouses/postgresql/renderexpr_test.go#L83), [SF](../../warehouses/snowflake/renderexpr_test.go#L77) | `"id" BETWEEN 5 AND 10` | `"ID" BETWEEN 5 AND 10` |
| Negazione ordinaria: [PG](../../warehouses/postgresql/renderexpr_test.go#L39), [SF](../../warehouses/snowflake/renderexpr_test.go#L33) | `"age" <> 18` | `"AGE" <> 18` |
| Riferimento a colonna: [PG](../../warehouses/postgresql/renderexpr_test.go#L43), [SF](../../warehouses/snowflake/renderexpr_test.go#L37) | `"a" = "b"` | `"A" = "B"` |
| Datetime con la precisione delle rispettive fixture: [PG](../../warehouses/postgresql/renderexpr_test.go#L67), [SF](../../warehouses/snowflake/renderexpr_test.go#L61) | `"timestamp" < '1900-01-02 23:32:11.940253'` | `"TIMESTAMP" < '1900-01-02 23:32:11.940253621'` |

La modalità strutturale deve continuare a gestire **tutti** i kind JSON, inclusi String, Number, Boolean e Null, sia per uguaglianza sia per negazione; non sostituirla con il dominio ristretto dei filtri pubblici. Aggiungere test di regressione dove mancano snapshot espliciti.

Tutti gli interi, inclusi int64, passano da Number API a `int`/`uint` interno: i literal SQL numerici restano identici per lo stesso valore Go, senza modifiche ai renderer degli interi. I float non vengono trasformati in decimal né sottoposti a 76/37; i decimal ordinari mantengono il serializer corrente, distinto dall'algoritmo JSON di Snowflake. Predicati ordinari, riferimenti colonna usati per matching e composizione con consensi mantengono le espressioni esistenti, comprese le differenze preesistenti della logica a tre valori su SQL NULL.

**Datetime:** i serializer attuali formattano `time.Time` senza chiamare `.UTC()`, con fino a sei decimali in PG e nove in Snowflake; conservarli. L'offset esplicito aggiunto alla String API evita di perdere l'istante, ma non autorizza normalizzazioni diverse prima del rendering. Restano separati eventuali problemi di normalizzazione temporale, collation ordinarie, presenza negli oggetti flatten e limiti degli array non JSON. [Serializer PG](../../warehouses/postgresql/renderexpr.go#L539), [Snowflake](../../warehouses/snowflake/renderexpr.go#L575).

## 9. Parsing e serializzazione nell'Admin

### 9.1 Stato attuale e problemi concreti

**C.** Il tipo API `FilterCondition.values` è `string[]` ed è riusato anche come stato dell'editor; `TransformedPipeline.filter` e `TransformedEventType.defaultFilter` usano lo stesso `Filter`. I controlli aggiungono righe incomplete e modificano stringhe. [Tipi API](../../admin/src/lib/api/types/pipeline.ts#L92), [modello editor](../../admin/src/lib/core/pipeline.ts#L274), [builder](../../admin/src/components/routes/PipelineWrapper/PipelineFilters.tsx#L59).

**C.** `parseText` elimina gli spazi esterni, interpreta apici singoli/doppi e gli escape delle virgolette, poi restituisce soltanto il contenuto. Non conserva se il testo era quotato. `formatString` non quota `"100"`, e rappresenta la String vuota come un controllo vuoto. Inoltre il suo escaping non coincide con quello del parser di filtri completo: nel controllo `\n` quotato significa la lettera `n` dopo l'escape, mentre il parser completo riconosce una newline. Non sostituire indiscriminatamente `parseText` con `JSON.parse`. [parseText](../../admin/src/utils/filters.ts#L224), [formatString](../../admin/src/utils/filters.ts#L16), [parseString del linguaggio completo](../../admin/src/utils/filters_parser.ts#L61).

**C.** Il solo chiamante attuale di `parseText` è `validateAndNormalizeFilterCondition`. La funzione calcola `values` parsati, ma chiama il validatore con `conditionValues` originali. Il caricamento applica `formatString` a ogni operando. Le anteprime chiamano il normalizzatore per validare, ne scartano il risultato e inviano un filtro dal quale sono soltanto rimosse le righe vuote. Questi percorsi devono diventare coerenti prima di inviare Number tipizzati. [Normalizzazione](../../admin/src/lib/core/pipeline.ts#L2109), [validazione valori](../../admin/src/lib/core/pipeline.ts#L2184), [caricamento](../../admin/src/lib/core/pipeline.ts#L628), [apertura anteprima](../../admin/src/components/routes/PipelineWrapper/PipelineTransformation.tsx#L867), [normalizedFilter](../../admin/src/components/routes/PipelineWrapper/PipelineTransformation.tsx#L1383).

### 9.2 Distinguere editor, modello tipizzato e trasporto

**P.** Non convertire tutto lo stato del controllo in `string | number`. Introdurre una rappresentazione esplicita del Number che conservi il testo, mantenendo le stringhe editoriali nei controlli:

```ts
interface FilterNumber {
    readonly kind: 'number';
    readonly text: string; // JSON Number valido, verificato senza Number(text).
}

type FilterValue = string | FilterNumber | boolean | null;

interface FilterCondition {
    property: string;
    operator: FilterOperator;
    values?: FilterValue[];
}

interface EditorFilterCondition {
    property: string;
    operator: FilterOperator | '';
    values?: string[];
}
```

Boolean e Null sono valori primitivi nel modello tipizzato, ammessi soltanto dopo validazione della membership JSON. `decodeFilter` conserva il kind dei valori ricevuti; la validazione dipendente da schema/operatore avviene nel preparatore, senza applicare 76/37 a tutti i Number della risposta. Anche i gruppi hanno una variante editoriale ricorsiva, con la stessa forma AND/OR. `FilterNumber` è un tipo **interno TypeScript**, non un oggetto da mandare come operando HTTP: il JSON effettivo deve contenere il numero nudo. I tipi di dominio restano in `admin/src/lib/`; evitare copie locali nei componenti. I visitor possono mantenere il loro comportamento con firme adeguate ai due alberi, senza cast fra essi.

Firme proposte, da collocare nelle utility dei filtri e nel livello di conversione pipeline:

```ts
const parseJSONFilterText = (input: string, operator: FilterOperator): FilterValue => /* ... */;
const formatJSONFilterValue = (value: FilterValue, operator: FilterOperator): string => /* ... */;
const prepareEditorFilter = (filter: EditorFilter | null, schema: ObjectType): Filter | null => /* ... */;
const toEditorFilter = (filter: Filter, schema: ObjectType): EditorFilter => /* ... */;
const decodeFilter = (value: unknown): Filter => /* valori restituiti dal decoder JSONbig */;
const toTransportFilter = (filter: Filter): TransportFilter => /* ... */;
const encodeFilter = (filter: Filter): string => /* JSONbig.stringify(toTransportFilter(filter)) */;
```

Le firme illustrano responsabilità, non implementazioni pronte. `prepareEditorFilter` deve riusare risoluzione della proprietà, rimozione delle righe incomplete e controlli oggi presenti nel normalizzatore; i vincoli dipendenti dal contesto, come le proprietà nascoste, rimangono applicati dai chiamanti. Il risultato tipizzato deve essere quello inviato, sia nel salvataggio sia nelle anteprime. Non ripetere il parsing nel client HTTP.

### 9.3 Evoluzione di `parseText` e regole di input

**P.** Preferire una funzione gemella `parseJSONFilterText(input, operator)` che riusi un tokenizer interno con `parseText`. Il tokenizer restituisce contenuto e provenienza `isQuoted`, accettando anche un contenuto quotato vuoto. Il wrapper `parseText` conserva firma e comportamento dei controlli non JSON; il wrapper JSON ammette `""` / `''` secondo A4. Non duplicare il ciclo degli escape. Un flag booleano cambierebbe il tipo di ritorno di `parseText` e non esprimerebbe il contesto membership.

Dopo trim degli spazi esterni:

1. Token esplicitamente quotato → String, anche se vuoto, numerico, `true`, `false` o `null`.
2. Solo in `is one of` / `is not one of` su JSON, token non quotato esattamente `true`, `false` o `null` → Boolean / Null.
3. Token non quotato conforme alla grammatica JSON Number → Number entro 76/37.
4. Altro token non quotato → String; controllo editoriale vuoto → condizione incompleta, mai String vuota implicita.

```text
^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$
```

Non riutilizzare `isFilterNumber` come recognizer JSON: accetta anche `+5`, `.5`, `1.` e zeri iniziali, e passa da `Number`. Un token JSON Number fuori 76/37 produce errore, non diventa String. Il contesto membership è esplicito; il parser non converte Boolean/Null negli altri operatori. [isFilterNumber](../../admin/src/utils/filters.ts#L12), [isDecimal corrente](../../admin/src/utils/filters.ts#L72).

| Input del controllo su JSON | Membership JSON | Altri operatori JSON, se compatibili |
|---|---|---|
| `100` | Number `100` | Number `100` |
| `100.50`, `1e3` | Number di valore esatto | Number di valore esatto |
| `"100"` oppure `'100'` | String `"100"` | String `"100"` |
| `EUR` | String `"EUR"` | String `"EUR"` |
| `" EUR "` | String `" EUR "` | String `" EUR "` |
| `00123` | String `"00123"` | String `"00123"` |
| `true`, `false`, `null` | Boolean true, Boolean false, Null | String `"true"`, `"false"`, `"null"` |
| `"true"`, `'false'`, `"null"` | String corrispondente | String corrispondente |
| `""` oppure `''` | String vuota | String vuota |
| Campo vuoto o soli spazi non quotati | Incompleto | Incompleto |
| `+100`, `.5`, `1.` | String corrispondente | String corrispondente |
| `1e76`, `1e-38` | Errore Number fuori limite | Errore Number fuori limite |

Per proprietà ordinarie il kind è determinato dallo schema: tutti gli int/year, inclusi signed/unsigned int64, producono Number tramite `FilterNumber`; float/decimal producono Number e String/UUID/IP/tempi String. Validare la sintassi intera **prima** di costruire BigNumber: non lasciare che `100.0` o `1e2` diventino `100` e aggirino A6. Conservare il token intero nel modello, verificarne il range esattamente sul testo o con BigInt e convertirlo direttamente in BigNumber per il trasporto, senza JavaScript Number come intermedio che determini il valore. I float conservano regole e limiti attuali, senza 76/37; decimal usa 76/37 senza P,S. Le eventuali normalizzazioni del testo editoriale ordinario in token JSON validi non devono alterare tali regole né i valori digitati.

### 9.4 Precisione JavaScript e librerie già disponibili

**C.** `call.ts` usa `JSONbig.stringify` per i body e `JSONbig.parse` per le risposte riuscite. Il vendor di `json-bigint` costruisce `BigNumber` dal **testo originale** per token lunghi più di 15 caratteri; per quelli più brevi usa Number. `alwaysParseAsBig` non cambia l'origine di quel ramo breve: costruisce un BigNumber dal Number già calcolato. `storeAsString` perderebbe la distinzione con le String reali; `useNativeBigInt` non rappresenta le frazioni. [call](../../admin/src/lib/api/call.ts#L19), [parse risposta](../../admin/src/lib/api/call.ts#L80), [opzioni](../../admin/node_modules_vendor/json-bigint/lib/parse.js#L84), [ramo numerico](../../admin/node_modules_vendor/json-bigint/lib/parse.js#L172).

**C.** `JSONbig.stringify` riconosce BigNumber prima di `toJSON` ed emette la sua rappresentazione come Number non quotato. Il costruttore `new BigNumber(text)` senza base conserva le cifre; specificare base 10 attiva invece un percorso che può arrotondare secondo `DECIMAL_PLACES`. Non modificare globalmente la configurazione BigNumber. [Serializer](../../admin/node_modules_vendor/json-bigint/lib/stringify.js#L218), [costruttore](../../admin/node_modules_vendor/bignumber.js/bignumber.js#L182), [ramo base esplicita](../../admin/node_modules_vendor/bignumber.js/bignumber.js#L255), [toJSON](../../admin/node_modules_vendor/bignumber.js/bignumber.js#L2760).

**P. Percorso di invio raccomandato:**

1. Conservare `FilterNumber.text` dal controllo, senza `Number(text)` / `parseFloat`.
2. Validare in base a schema e operatore: sintassi/range per int e year, regole correnti per float, grammatica e 76/37 per decimal e Number JSON. Solo per questi ultimi si può contare la rappresentabilità da coefficiente ed esponente senza eseguire aritmetica decimale: eliminare gli zeri finali del coefficiente, calcolare l'esponente effettivo, derivare precisione e scala con le stesse regole di `Decimal.PrecisionScale`. Per zero usare 1/0. Limitare il lavoro sugli esponenti prima di costruire potenze o stringhe espanse.
3. In `toTransportFilter` sostituire soltanto i `FilterNumber` con veri `BigNumber` costruiti dal testo, dopo la validazione, anche per tutti gli interi. String, Boolean e Null restano tali. `TransportFilter` ha valori `string | BigNumber | boolean | null`; int64 segue il ramo BigNumber e diventa Number nel JSON HTTP. Non usare un adattatore numerico globale sui token che sembrano numeri.
4. Serializzare con `JSONbig.stringify` nei body e nelle query string. Non introdurre placeholder da sostituire nel JSON, injection di token raw o falsi oggetti che imitano `_isBigNumber`.

Importare direttamente BigNumber richiederà dichiarare la dipendenza diretta e aggiornare la risoluzione vendor nella futura implementazione: oggi è transitiva di `json-bigint`, non un import pubblico già configurato per l'Admin. Questo è riuso della dipendenza presente, non richiesta di introdurre un secondo motore decimale. [package.json](../../admin/package.json#L17), [risoluzione vendor](../../admin/node_modules_vendor/resolve.json#L2152).

Per rendere verificabile il punto 2: dopo aver rimosso gli zeri iniziali e finali dal coefficiente non nullo, siano k il suo numero di cifre ed e l'esponente effettivo, già corretto per la parte frazionaria e per gli zeri finali rimossi. Allora `scale = max(-e, 0)` e `precision = max(k + e, 0) + scale`. Si calcolano soltanto lunghezze ed esponenti interi controllati; nessuna potenza né conversione del coefficiente in Number. Per esempio `0.005` ha k=1, e=-3 e quindi precision=3, scale=3.

**P. Percorso di caricamento:** mantenere il decoder generale attuale e normalizzare i soli campi `Filter` delle risposte prima di passarli a **qualsiasi consumatore del modello tipizzato**, compresi griglia ed editor. String rimane String; BigNumber diventa `FilterNumber` dal suo testo esatto. I Number prodotti dal ramo breve del decoder si convertono al testo numerico senza operazioni aritmetiche. Questa eccezione riguarda esclusivamente valori ricevuti dal decoder noto, non i numeri digitati o arbitrari Number costruiti dai chiamanti.

Per gli interi, il ramo breve contiene soltanto valori rappresentabili esattamente come Number; quelli oltre `Number.MAX_SAFE_INTEGER` hanno token più lunghi di 15 caratteri e il parser restituisce `new BigNumber(string)` dal testo originale. Il vendor calcola anche `+string` prima del dispatch, ma lo usa per il controllo di finitezza: il risultato potenzialmente arrotondato non alimenta il BigNumber. `decodeFilter` deve quindi derivare `FilterNumber.text` dal BigNumber ricevuto, senza convertirlo in Number. [Percorso effettivo](../../admin/node_modules_vendor/json-bigint/lib/parse.js#L204).

Il trasporto degli interi deve emettere token **senza frazione né esponente**, coerentemente con A6. Gli estremi a 64 bit hanno al massimo 20 cifre di modulo: il vendor BigNumber mantiene questa notazione con la configurazione corrente, che passa alla notazione esponenziale da esponente 21. Non cambiare la configurazione globale; testare il token JSON effettivo, non solo l'uguaglianza numerica del BigNumber. Per esempio il valore `FilterNumber` `{kind: 'number', text: '9223372036854775807'}` passa a `new BigNumber(text)` e produce `values: [9223372036854775807]`, senza String o wrapper. [Configurazione](../../admin/node_modules_vendor/bignumber.js/bignumber.js#L104), [formattazione](../../admin/node_modules_vendor/bignumber.js/bignumber.js#L2757), [serializer](../../admin/node_modules_vendor/json-bigint/lib/stringify.js#L218).

**C/P. Punto concreto di ingresso:** le pipeline arrivano in `Connection.pipelines`. `Connections.find` e `Connections.get` restituiscono oggi direttamente gli oggetti decodificati; `TransformedConnection` conserva l'array e `PipelinesGrid` legge il filtro senza passare da `transformPipeline`. Quindi applicare `decodeFilter` ai `c.pipelines[].filter` presenti in entrambi i metodi API, prima del cast a `Connection`. Non basta aggiungerlo nell'apertura dell'editor. Non trasformare il `defaultFilter` testuale degli event type come se fosse un albero JSON; quello continua nel parser del linguaggio completo. [Tipo Connection](../../admin/src/lib/api/types/connection.ts#L32), [find/get](../../admin/src/lib/api/api.ts#L256), [TransformedConnection](../../admin/src/lib/core/connection.ts#L61), [griglia](../../admin/src/components/routes/ConnectionPipelines/PipelinesGrid.tsx#L244).

**D.** Nel dominio valido 76/37, i token del ramo breve hanno al massimo 15 cifre significative e non possono richiedere esponenti non nulli fuori dal normale intervallo binario: la forma decimale breve può essere ricostruita semanticamente. I numeri che richiedono più cifre passano già dal testo a BigNumber. Non è dunque corretto affermare che il decoder presente arrotondi necessariamente i Number ad alta precisione dei filtri. È però necessario verificare questo confine con test differenziali prima del rilascio, senza cambiare globalmente tutti i numeri delle risposte in oggetti.

Nell'analisi precedente un esperimento Node ha caricato parser e serializer vendorizzati in memoria, senza scrivere file. Su 100.000 token generati, 59.941 rispettavano 76/37 secondo un controllo decimale esatto; di questi, 9.508 passavano dal ramo Number e 50.433 dal ramo BigNumber. Il confronto del valore prima e dopo parse/stringify con BigNumber dal token originale non ha rilevato differenze. Verificati separatamente `100`, `100.0`, `1e2`, `1e75`, `1e-37`, `9007199254740993` e una frazione a scala 37. È evidenza limitata, non una prova esaustiva né esecuzione della suite Admin. La conservazione della grafia `100.50` rispetto a `100.5` non è richiesta: devono restare valore e kind.

È stato verificato anche il trasporto di `[BigNumber("9007199254740993"), "9007199254740993"]`: JSONbig emette `[9007199254740993,"9007199254740993"]`; `JSON.stringify` nativo emette due String. Il problema certo in questo percorso è quindi anche la perdita del kind, oltre al rischio di arrotondamento quando si costruiscono prima Number JavaScript.

**C. Verifica mirata di questa revisione:** eseguito in memoria un round-trip con parser/serializer vendorizzati per `-9223372036854775808`, `9223372036854775807`, `0`, `18446744073709551615`, `9007199254740991`, `9007199254740992`, `9007199254740993` e `-9007199254740993`. Tutti gli otto casi hanno prodotto il Number JSON intero esatto e lo hanno conservato dopo parse/stringify; i token lunghi sono stati riletti come BigNumber. Non sono stati creati file né eseguiti test del nuovo editor, che resta da implementare. Questa prova verifica il percorso vendor; §13 richiede il round-trip completo nell'Admin.

Se la verifica differenziale trovasse un caso valido non conservato, è bloccante correggere il decoder degli operandi mediante accesso ai token originali; un reviver dopo la conversione a Number non può recuperare cifre perdute. Non adottare come soluzione un fallback che converta Number in String nel JSON HTTP. Il controllo 76/37 di questa verifica riguarda decimal e Number JSON, non float: anche una risposta con soglia float64 `1e100` deve restare caricabile. Boolean false e Null passano dal decoder senza trasformazioni numeriche.

### 9.5 HTTP, anteprime e round-trip

Aggiornare tutti i punti che serializzano un filtro:

| Percorso | Comportamento corrente | Intervento |
|---|---|---|
| Lista/dettaglio connessioni | Pipeline annidate restituite dopo il parsing generale | Decodificare i filtri nel modello tipizzato in `find` e `get`, prima di griglia/editor |
| Creazione/aggiornamento pipeline | Body passato a `call` con JSONbig | Convertire il filtro tipizzato in trasporto prima della chiamata |
| Listener eventi per anteprima | Filtro nel body | Preparare il filtro editoriale e adattarlo al trasporto |
| Utenti applicativi | `JSON.stringify(filter)` in query | `encodeFilter` |
| Ricerca profili | `JSON.stringify(filter)` in query | `encodeFilter` |
| Ricerca eventi del profilo | Filtro String su `kpid` costruito nel client | Usare lo stesso encoder; il valore UUID resta String |

Riferimenti: [connections.find/get](../../admin/src/lib/api/api.ts#L256), [create/update](../../admin/src/lib/api/api.ts#L414), [apiUsers](../../admin/src/lib/api/api.ts#L482), [listener](../../admin/src/lib/api/api.ts#L537), [profiles.find](../../admin/src/lib/api/api.ts#L577), [eventi](../../admin/src/lib/api/api.ts#L617).

`JSON.stringify` nativo non è adatto: su BigNumber può produrre una String numerica attraverso `toJSON`; su `FilterNumber` produrrebbe l'oggetto interno. Anche un body già supportato da JSONbig deve ricevere l'albero di trasporto corretto.

Sul caricamento schema-aware `Filter → EditorFilter`, un Number JSON si mostra non quotato; una String JSON si mostra **sempre quotata**, inclusi `"100"`, `"true"`, `"false"`, `"null"` e `""`. Boolean e Null si mostrano non quotati nella membership. Tutti gli interi, inclusi int64, si mostrano come testo intero e tornano nel payload come Number attraverso `FilterNumber`/BigNumber, senza arrotondamenti. Il formatter deve essere l'inverso del parser del **controllo** per spazi, backslash e virgolette. Non usare direttamente il quoting JSON dove `parseText` ha regole di escape diverse.

La proprietà richiesta è:

```text
prepareEditorFilter(toEditorFilter(loadedFilter, schema), schema)
    ≡ loadedFilter per kind e valore semantico degli operandi
```

Vale anche per liste miste come `[100, "100", true, false, null, "true", "null", ""]`, spazi, apici, backslash, newline e numeri a 76 cifre. Non scartare false o Null con controlli di truthiness, `filter(Boolean)` o conversioni del tipo `value ?? ""`: diventerebbero valori mancanti o String vuote. Il pruning deve guardare il **testo del controllo**, non la verità del valore tipizzato. Non è richiesta identità byte-per-byte del token Number. Per UUID/IP e formati temporali preservare le normalizzazioni già previste; per datetime conservare l'offset necessario al round-trip senza introdurre normalizzazione SQL.

### 9.6 Parser completo, serializer e default dei connettori

Il linguaggio completo di `parseFilter` legge String quotate con escape JS, token numerici permissivi e `true`/`false` come testi; oggi non riconosce un token Null. Perde le virgolette e `serializeFilter` decide se quotare riconoscendo il testo numerico. Il round-trip attuale fra stringhe non verifica i nuovi kind. [parseValue](../../admin/src/utils/filters_parser.ts#L123), [parseFilter](../../admin/src/utils/filters_parser.ts#L224), [serializeFilter](../../admin/src/utils/filters.ts#L258), [test](../../admin/src/utils/filters_parser_test.ts#L341).

**P.** Conservare provenienza quotata/non quotata e testo numerico durante il parsing lessicale; riusare queste informazioni nel passaggio **schema-aware** all'`EditorFilter`. Per esempio, la conversione completa può ricevere lo schema come `parseFilter(source, schema)`. Una String quotata di una proprietà JSON viene riformattata come String quotata nel controllo; Number resta non quotato; `true`/`false`/`null` non quotati assumono i nuovi kind soltanto per membership JSON. Per i tipi ordinari, compreso int64, schema e tipo del controllo determinano la rappresentazione editoriale senza aggiungere virgolette visibili spurie. Non usare BigNumber per normalizzare token int/year prima della verifica sintattica.

Aggiungere il riconoscimento lessicale di `null`, mantenendo la distinzione da `"null"`; gli altri operatori JSON lo interpretano come testo secondo §9.3. Preservare gli escape JS già supportati dal linguaggio completo, traducendoli nel contenuto corretto del controllo, senza confonderli con gli escape più semplici di `parseText`.

`serializeFilter` su un filtro tipizzato quota tutte le String, anche numeriche e booleane apparenti, scrive i Number dal testo verificato e Boolean/Null come token nudi esclusivamente nel contesto ammesso. Deve ricevere il modello tipizzato, non dedurre i tipi dal draft. Per eventuali visualizzazioni del draft mantenere un percorso esplicito. Il criterio completo diventa:

```text
prepareEditorFilter(parseFilter(serializeFilter(filter), schema), schema)
    ≡ filter per kind e valore semantico
```

Il default di un connettore può arrivare prima della risoluzione dello schema: mantenere sorgente o token fino al passaggio schema-aware, senza perdere le virgolette né inferire Boolean/Null soltanto dall'operatore senza conoscere il tipo della proprietà. Adeguare il punto attuale di `useConnection` e il passaggio al modello pipeline; non modificare il formato dei default Go. [Caricamento](../../admin/src/components/routes/ConnectionWrapper/useConnection.ts#L59), [costruzione default](../../admin/src/lib/core/pipeline.ts#L1185).

I default individuati in Google Analytics, Mixpanel e PostHog confrontano `type`/`event` con String quotate e usano unari JSON. Devono mantenere il significato, così come il default Admin `identify OR traits is not empty`; non trasformare nomi di evento in Number. [Definizione](../../connectors/applications.go#L267), [Google Analytics](../../connectors/googleanalytics/event_types.go#L102), [Mixpanel](../../connectors/mixpanel/mixpanel.go#L84), [PostHog](../../connectors/posthog/posthog.go#L86).

## 10. Modifiche ai chiamanti e alle integrazioni

### 10.1 Inventario Go

La ricerca ha incluso costruttori, `Values`, conversioni, persistenza e i due `JSONConditionValue`, non soltanto `filters.go`. La tabella distingue i filtri pubblici dagli alberi interni: non sostituire indiscriminatamente ogni `[]any{string}` con `json.Value`.

| Componente / punti verificati | Intervento necessario |
|---|---|
| [core/filters.go](../../core/filters.go#L101) e [filters_test.go](../../core/filters_test.go#L29) | Campo pubblico, decoder, encoder, preparazione, inverse conversion, commenti e fixture per ogni tipo |
| [Connection.ApplicationUsers](../../core/connection.go#L225) | Usare il risultato preparato contro lo schema richiesto; anteprima Go |
| [Workspace.CreateEventListener](../../core/workspace.go#L766), [Events](../../core/workspace.go#L914), [Profiles](../../core/workspace.go#L1251) | Preparare una volta con lo schema appropriato e passare lo stesso `Where` al percorso Go/SQL |
| [validatePipelineToSet](../../core/pipeline_validate.go#L74), [test](../../core/pipeline_validate_test.go) | Rendere disponibile il risultato preparato senza perdere la raccolta delle proprietà usate |
| [Connection.CreatePipeline](../../core/connection.go#L400), [Pipeline.Update](../../core/pipeline.go#L778) | Riutilizzare preparazione e schema effettivo; serializzare il nuovo `Where` |
| [Pipeline.fromState](../../core/pipeline.go#L1148) | Risposta API con Number e String corretti |
| [cmd/workspaces.go, filtro kpid](../../cmd/workspaces.go#L849) | Costruttore pubblico diretto: `[]json.Value` con UUID quotato correttamente |
| [cmd/connection.go](../../cmd/connection.go#L38), [cmd/pipeline.go](../../cmd/pipeline.go#L127), [cmd/workspaces.go](../../cmd/workspaces.go#L878) | Verificare body e query-string decode; nessuna conversione numerica schema-aware negli handler |
| [state/where.go](../../core/internal/state/where.go#L98), [where_test.go](../../core/internal/state/where_test.go#L20), [load.go](../../core/internal/state/load.go#L608), [keep.go](../../core/internal/state/keep.go#L629) | Persistenza, ripristino, uguaglianza, lettura tramite notifiche e test relativi |
| [filters/applies.go](../../core/internal/filters/applies.go#L106) e [test](../../core/internal/filters/applies_test.go) | Semantica JSON tipizzata; conservare operatori ordinari |
| [datastore/query.go](../../core/internal/datastore/query.go#L99), [query_test.go](../../core/internal/datastore/query_test.go), [query_json_test.go](../../core/internal/datastore/query_json_test.go#L34) | Rimuovere adattatore duale, impostare la modalità JSON tipizzata e verificare valori raw e confronti differenziali |
| [warehouses.go](../../warehouses/warehouses.go#L854), renderer e test PostgreSQL/Snowflake | Rimuovere il duale, distinguere confronto tipizzato e strutturale; aggiornare `renderexpr_test.go` e `jsonexpr_test.go` in entrambi i package |
| [test/warehouseexpr/json.go](../../test/warehouseexpr/json.go#L52), [json_test.go](../../test/warehouseexpr/json_test.go#L17) | Sostituire l'helper duale `Operand` con fixture S/N e membership S/N/B/Z; aggiornare risultati attesi con un oracolo indipendente e i test dell'helper |
| [test/krenalistester/support_types.go](../../test/krenalistester/support_types.go#L126), [support.go](../../test/krenalistester/support.go#L410), [support_types_test.go](../../test/krenalistester/support_types_test.go#L12) | Modello di test duplicato, default e filtro kpid; preservare l'attuale `values:null` dei suoi unari, che il server accetta |
| [anonymous_not_anonymous_test.go](../../test/anonymous_not_anonymous_test.go), [source_app_users_filtering_test.go](../../test/source_app_users_filtering_test.go), [source_file_storage_users_filtering_test.go](../../test/source_file_storage_users_filtering_test.go) | Costruttori pubblici di fixture: codificare String come String e soglie numeriche come Number |

Esempio per un chiamante Go diretto, con una costante String JSON:

```go
condition := &core.FilterCondition{
    Property: "type",
    Operator: core.OpIs,
    Values:   []json.Value{json.Value("\"identify\"")},
}
```

Per String dinamiche usare `json.Quote` o il marshal esistente, gestendone l'errore; non concatenare virgolette attorno al contenuto. Per Number da `decimal.Decimal` usare `MarshalJSON`. La preparazione deve comunque validare il risultato, perché questi costruttori non passano dall'unmarshal HTTP.

I `state.Where` costruiti direttamente per filtri su normali UUID/String devono continuare a usare String Go, per esempio in [Workspace.Attributes](../../core/workspace.go#L253), [Workspace.Identities](../../core/workspace.go#L1031) e [Connection.Identities](../../core/connection.go#L1087). Solo un operando di proprietà JSON è `json.Value` nell'albero interno.

### 10.2 Pipeline, ingestion, dispatch, ricerca ed export

Non cambia il punto in cui il filtro viene eseguito:

- Collector: ingestion eventi, estrazione identità da eventi e dispatch usano `filters.Applies`. [collector.go](../../core/internal/collector/collector.go#L850).
- Import utenti: applicazione su attributi normalizzati. [run_import.go](../../core/run_import.go#L180).
- Listener live e anteprime applicative: valutazione Go. [observer.go](../../core/internal/collector/observer.go#L177), [application.go](../../core/internal/connections/application.go#L727).
- Ricerca profili/eventi e selezione per export: compilazione SQL dello stesso `Where`. [run_export.go](../../core/run_export.go#L65), [ProfileRecords](../../core/internal/datastore/store.go#L458), [composizione records](../../core/internal/datastore/records.go#L48).

Nell'export, consensi, matching destinazione, modalità create/update e proiezione delle colonne rimangono composizione del livello datastore. Il nuovo operando non deve cambiare condizioni di join, colonne richieste, conteggi o paginazione. Un filtro su proprietà ordinaria ha lo stesso valore Go di prima; un filtro JSON ha un dominio più esplicito, e quindi può intenzionalmente selezionare risultati diversi dal vecchio duale.

La raccolta delle proprietà usate è un risultato necessario della preparazione: `validateFilterGroup` restituisce la proprietà base quando il percorso continua nel JSON; il validatore pipeline la usa nel controllo delle proprietà di input. L'Admin aggiunge la proprietà base allo schema in `transformInPipelineToSet`. Conservare entrambi i comportamenti; tipizzare il valore non deve perdere la colonna JSON necessaria alla pipeline. [paths](../../core/filters.go#L978), [uso input paths](../../core/pipeline_validate.go#L160), [Admin addPropertyToSchema](../../admin/src/lib/core/pipeline.ts#L965).

### 10.3 Inventario Admin e controlli di chiusura

Oltre ai file di §9, coinvolgere [visitor e pruning](../../admin/src/utils/filters.ts#L78), [PipelineFilters](../../admin/src/components/routes/PipelineWrapper/PipelineFilters.tsx#L67), [PipelineTransformation](../../admin/src/components/routes/PipelineWrapper/PipelineTransformation.tsx#L867), [PipelinesGrid](../../admin/src/components/routes/ConnectionPipelines/PipelinesGrid.tsx#L246), [modello connessione](../../admin/src/lib/core/connection.ts#L26), [tipo risposta Connection](../../admin/src/lib/api/types/connection.ts#L32), [useConnection](../../admin/src/components/routes/ConnectionWrapper/useConnection.ts#L59), [test parser](../../admin/src/utils/filters_parser_test.ts), [helper UI](../../admin/tests/utils.ts), [test pipeline](../../admin/tests/pipelines.spec.ts).

Prima di considerare completa l'implementazione, ripetere la ricerca di `FilterCondition`, `.Values`/`.values` nei contesti filtro, `JSONConditionValue`, `parseFilter`, `parseText`, `serializeFilter`, `convertFilterToWhere`, `convertWhereToFilter`, `validateFilter` e `JSON.stringify(filter)`, escludendo vendor e distinguendo nomi omonimi non relativi ai filtri. Il risultato atteso non è zero per `Values`: è zero per il duale rimosso e nessun percorso filtro che assuma ancora `[]string` o serializzazione numerica nativa.

## 11. Gestione dei filtri precedentemente persistiti

**Avvertenza per il futuro rilascio.** Il nuovo formato non è compatibile con tutti i filtri persistiti: gli operandi JSON sono oggi String dalle quali il loader ricostruisce il duale. Con il nuovo decoder alcuni payload possono essere rifiutati, altri restare leggibili ma cambiare significato. Gli interi persistiti sono invece già Number, anche per int64, e conservano questa rappresentazione. Il problema degli operandi JSON riguarda anche i payload contenuti nelle notifiche di stato. [Writer attuale](../../core/internal/state/where.go#L301), [loader JSON](../../core/internal/state/where.go#L376), [load](../../core/internal/state/load.go#L608), [notifiche](../../core/internal/state/keep.go#L549).

Prima di un futuro rilascio sarà necessario valutare i filtri effettivamente presenti e il loro trattamento. Questa specifica si limita ad aggiornare marshal/unmarshal al nuovo formato: non progetta convertitori, migrazioni, procedure operative o sistemi di compatibilità. Nessun versionamento del formato o percorso legacy permanente.

## 12. Ottimizzazioni eventuali

Il duale precedente a Typed Filter Values evitava di parsare la soglia numerica JSON a ogni valutazione. La sua rimozione introduce lavoro ripetuto, ma non cambia l'obiettivo funzionale.

| Opzione | Costo / vantaggio | Valutazione |
|---|---|---|
| Parsing durante ogni valutazione | Ogni confronto Number rilegge la soglia; nessun nuovo stato o invalidazione | **Raccomandata inizialmente**: modifica più semplice e verificabile |
| Preparazione preventiva del valutatore | Compila un albero separato con Number costanti già parsati | Utile soltanto se il costo risulta significativo; `state.Where` conserva comunque `json.Value` |
| Cache della condizione compilata | Ammortizza il parsing nei percorsi ripetuti | Richiede proprietà di immutabilità, durata e concorrenza esplicite; non usare chiavi basate su slice o puntatori mutabili senza contratto |

Il parsing della **soglia costante** può essere evitato da una preparazione futura; quello del **numero letto nel record JSON** rimane necessario se il record contiene solo byte JSON. Non attribuire alla cache dell'operando un risparmio sul parsing dei dati. Non ripristinare `JSONConditionValue` per anticipare questa ottimizzazione.

La validazione/preparazione core resta comunque una sola volta per richiesta o aggiornamento; l'eventuale parsing runtime ripetuto non giustifica duplicare la validazione core. Nel rendering SQL si può preparare localmente il decimal dell'operando durante la singola compilazione, senza cache persistente. La progettazione iniziale non includeva benchmark; le misure successive all'implementazione e il debito tecnico residuo sono riportati in §12.1.

### 12.1 Debito tecnico noto: prestazioni della membership JSON

**Aggiornamento successivo all'implementazione, 25 ottobre 2026. Stato: ottimizzazione intenzionalmente rinviata. Nessun problema di correttezza funzionale individuato.** Questa nota registra un costo misurato e non modifica le decisioni funzionali A1–A6.

Gli operandi JSON Number sono conservati come `json.Value` in `state.WhereCondition.Values`. Durante `is one of` e `is not one of`, il valutatore interpreta nuovamente come `decimal.Decimal` ciascun operando Number necessario al confronto. Questo lavoro sui valori costanti si ripete per ogni record valutato; il costo aumenta con il numero di candidati esaminati prima di un eventuale match e con la complessità dei numeri. [Rappresentazione interna](../../core/internal/state/where.go#L99), [opIsIn](../../core/internal/filters/applies.go#L291).

La prima ottimizzazione prepara il valore del record al massimo una volta, al primo candidato di kind compatibile, evita la decodifica delle String senza escape e riusa un buffer locale per gli operandi String con escape. Ha eliminato molti parsing e allocazioni inutili, ma non il parsing ripetuto degli operandi Number tra valutazioni di record diversi. I confronti rimangono esatti tramite `decimal.Decimal`, senza passaggi attraverso `float64`; `state.Where` continua a conservare `json.Value`.

**Misure di riferimento.** Microbenchmark su **Apple M2, Go 1.27.2, darwin/arm64**, mediane di cinque esecuzioni con `-cpu=1 -benchtime=100ms -count=5`: membership con **128 candidati e nessun match**, preparazione del filtro esclusa dal tempo misurato. La baseline precedente a Typed Filter Values è il commit `82e7be3a1939340cfc3374cf9f8458a63f21a43b`, eseguito in una copia temporanea con gli operandi preparati nella rappresentazione precedente, fuori dalla regione misurata. Il confronto riguarda i casi String/Number omogenei compatibili con entrambe le rappresentazioni.

| Caso | Baseline precedente | Implementazione dopo la prima ottimizzazione |
|---|---:|---:|
| String senza escape | 0,318 µs/op | 0,803 µs/op |
| String con escape | 0,344 µs/op | 3,842 µs/op |
| Number piccoli | 1,388 µs/op | 5,733 µs/op |
| Number grandi | 2,071 µs/op | 12,946 µs/op |
| Number precisione 76, scala 37 | 3,906 µs/op | 174,061 µs/op |

Questi risultati misurano soltanto `opIsIn`, condiviso dai due operatori di membership. **Non sono misure del throughput complessivo di Krenalis**, né valori attesi su qualsiasi macchina: dipendono da hardware, versione Go, ambiente, carico concorrente e fixture. In particolare, non quantificano il costo complessivo di preparazione del filtro, lettura del record o esecuzione delle pipeline.

**Impatto potenziale.** Il costo residuo può incidere soprattutto sui percorsi che valutano molti record, come ingestion e import, quando usano membership JSON numerose o numeri ad alta precisione. L'impatto sulle prestazioni reali dell'applicazione non è stato ancora misurato. I risultati motivano la registrazione del debito tecnico, senza stabilire da soli una priorità operativa o una soluzione architetturale.

**Benchmark conservato e riproduzione.** Il file [core/internal/filters/applies_bench_test.go](../../core/internal/filters/applies_bench_test.go) contiene `BenchmarkOpIsInJSON`: **11 famiglie × 4 dimensioni × 3 posizioni = 132 scenari**. Le dimensioni sono 1, 8, 32 e 128 operandi; le posizioni sono `NoMatch`, `First` e `Last`. Le famiglie coprono String con e senza escape, Number piccoli/grandi/a precisione 76 e scala 37, liste miste con Boolean/Null e liste di kind incompatibile. Le fixture String omogenee condividono prefisso e lunghezza decodificata; la preparazione delle fixture avviene fuori dal ciclo misurato.

Dalla radice del repository, con la toolchain prevista da `go.mod`:

```sh
go test ./core/internal/filters -run '^$' -bench '^BenchmarkOpIsInJSON$' -benchmem -benchtime=100ms -count=5 -cpu=1
```

L'output riporta `ns/op`, `B/op` e `allocs/op`: tempo, byte allocati e numero di allocazioni per valutazione. `b.ReportAllocs()` e `-benchmem` mantengono visibili le allocazioni. Con cinque esecuzioni si può confrontare la mediana di ciascuno scenario; i tempi in tabella sono convertiti da nanosecondi a microsecondi.

Per verificare rapidamente l'eseguibilità di tutti i casi insieme ai test del package, senza ottenere una misura prestazionale confrontabile:

```sh
go test ./core/internal/filters -bench '^BenchmarkOpIsInJSON$' -benchmem -benchtime=1x -count=1 -cpu=1
```

Per confrontare versioni future, eseguire il comando di misura in checkout o worktree isolati delle due revisioni, sulla stessa macchina, con gli stessi flag, toolchain e file di benchmark. Salvare l'output separatamente, per esempio con `> /tmp/json-membership-before.txt` e `> /tmp/json-membership-after.txt`, registrando revisioni e ambiente; confrontare gli stessi nomi di scenario, sia per il tempo sia per le allocazioni. Se una revisione richiede una diversa costruzione delle fixture, documentare l'adattamento e mantenerlo fuori dalla regione misurata. Non confrontare direttamente casi con semantica diversa, come la membership Boolean/Null e le corrispondenti liste della vecchia rappresentazione. I log temporanei completi non devono essere conservati nel repository.

Il benchmark resta uno strumento di misura: le sue asserzioni verificano il risultato logico, **non soglie obbligatorie di tempo o allocazioni in CI**. Non sono richieste nuove dipendenze o infrastrutture.

**Possibile intervento futuro, non approvato.** Valutare una preparazione degli operandi JSON una sola volta per filtro, così da riusare le conversioni dei valori costanti tra record. Un valutatore preparato separato da `state.Where`, o un meccanismo equivalente, è una possibilità da esaminare dopo aver verificato come i filtri vengono riutilizzati, la durata e l'immutabilità degli operandi, l'eventuale invalidazione e la sicurezza rispetto alla concorrenza. Una simile preparazione non eliminerebbe il parsing del valore numerico letto nel record JSON. Nessuna di queste alternative è una decisione architetturale già presa; ulteriori ottimizzazioni sono intenzionalmente rinviate.

## 13. Piano di test

I casi seguenti sono **da implementare successivamente**. I risultati già disponibili in §13.7 riguardano la baseline; nessun test della nuova funzionalità è stato eseguito.

### 13.1 API, modello pubblico e conversione

Estendere `core/filters_test.go` e verificare il decoder specifico:

- Marshal/unmarshal di tutti i tipi di §3.2: Number per tutti gli int signed/unsigned, inclusi 64 bit, year, float/decimal; String per tempi/UUID/IP; operatori con 0/1/2/molti valori.
- Tutti gli int: accettare Number e rifiutare String numeriche, anche per int64. Verificare gli estremi signed `-9223372036854775808` / `9223372036854775807` e unsigned `0` / `18446744073709551615`, i valori immediatamente esterni e unsigned negativi; `9007199254740993` resta esatto. Soglie oltre il range della bit size più piccola restano valide entro il range signed/unsigned 64 bit. Per int/year rifiutare `100.0` / `1e2` prima della normalizzazione; conservare il range year 1–9999.
- Su JSON, Number 100 distinto da String `"100"`; membership mista `[100, "100", true, false, null, "true", "null", ""]` ammessa. Boolean/Null rifiutati dagli altri operatori con valori; Array/Object sempre rifiutati come operandi pubblici.
- Distinguere omissione, `values: null`, `[]`, `[null]` e slice raw nil costruita in Go. Gli unari accettano solo slice nil; `[null]` è un operando valido per membership, non una String vuota.
- A1: estremi omogenei S/S oppure N/N validi, misti rifiutati. A3/A5: contains S/N, prefisso/suffisso solo S. A4: String JSON vuota valida, controlli String ordinari invariati.
- Raw costruiti in Go: nil, vuoto, JSON invalido, documenti multipli, UTF-8 invalido, numeri non JSON; nessun panic. Copia dei byte rispetto a buffer/chiamante mutabili.
- Unmarshal ripetuto nella stessa variabile `Filter` e `FilterCondition` standalone, con slice precedentemente non vuote.
- Solo decimal/Number JSON: precisione 76 e scala 37 ai limiti, 77/38 rifiutati, zero/zeri finali ed esponenti; `decimal(5,2)` con soglia `1000.00` valida.
- Float: regole correnti, `1e100` valido per float64, overflow per float32, rifiuto di NaN/Infinity String e valori non rappresentabili. Nessuna verifica 76/37 condivisa con i decimal.
- Zero con esponenti estremi dopo la correzione separata, sintassi invalida ancora rifiutata dopo overflow. `1e100` rifiutato soltanto nel dominio decimal/JSON, non genericamente come Number.
- Stessi risultati di preparazione da Go e HTTP; proprietà uniche/ordinate e base JSON conservate, limiti 4/100 invariati. Decoder condiviso core/state coerente su tipo, operatore e limiti, senza P,S della proprietà decimal.

Nei test HTTP verificare ordine `filter`/`inSchema`, create/update pipeline, query string e classificazione degli errori core. Conservare la distinzione fra marshal strutturale e validazione semantica.

### 13.2 Persistenza e uguaglianza

Estendere `core/internal/state/where_test.go`:

- `MarshalWhere → unmarshalWhere → convertWhereToFilter` conserva kind/valore: tutti gli interi sono scritti/letti come Number, con estremi signed/unsigned 64 bit esatti e senza distinzione di bit size nel serializer; float senza 76/37, JSON S/N/B/Z diretti.
- `[null]` rimane un operando Null nel nuovo formato; `values: null` rimane assenza di operandi. Nessun secondo documento JSON dentro una String e nessun fallback del duale.
- Equal: Number `100`/`100.0`/`1e2` equivalenti; String `"100"` diversa; escape String equivalenti; Boolean true/false distinti e non equivalenti a Number/Null/String; Null uguale a Null.
- Gruppi/valori ordinati, nessuna equivalenza logica aggiunta; nil/empty slice secondo il contratto esistente.
- Caricamento da `pipelines` e notifiche **nel nuovo formato**, errori sui valori invalidi. Nessun test di convertitori o compatibilità del formato precedente in questo piano.
- Datetime: offset/frazioni preservano l'istante nel round-trip API; writer/loader persistito mantengono la politica UTC esistente. Snapshot separati del SQL diretto e del SQL dopo caricamento per assicurare che ciascun percorso conservi il comportamento precedente, senza introdurre una normalizzazione comune.
- Normalizzazione decimal/Number JSON, anche per zero con esponente estremo e molti zeri finali; persistenza reale nel JSONB di Krenalis senza presumere che tutti i parser SQL accettino ogni grafia Go.

### 13.3 Runtime Go

Ampliare `core/internal/filters/applies_test.go` con risultati attesi indipendenti dal nuovo codice:

- Uguaglianza `100`/`100.0`/`1e2`, ordinamenti esatti, numeri grandi vicini e scala 37; String `"9"` incompatibile con Number 100, maggiore della String `"100"` lessicograficamente.
- Membership dei quattro kind, inclusi true, false e Null. Missing sempre false al positivo anche con `[null]`; Null presente true solo se incluso. Coprire sia `json.Value("null")` sia nil presente nelle mappe Go.
- String `"true"`/`"false"`/`"null"`, Number 0/1 e Boolean non si confondono. Liste miste e duplicati; Array/Object incompatibili.
- Negazioni come complemento su dati presenti, assenti, null e kind incompatibili; AND/OR annidati e short-circuit.
- Array misti, array annidati non attraversati, `array(json)`; Number solo elementi numerici, String sottostringa/elementi String, nessuna appartenenza Boolean/Null tramite contains.
- Between omogenei inclusivi/invertiti, String vuota e relativi effetti su contiene/prefisso/suffisso. Unari true/false/null/esistenza/vuoto invariati.

Mantenere i test dei tipi ordinari e della lettura presenza/valore; non cambiare implicitamente semantica temporale o negazioni ordinarie.

### 13.4 PostgreSQL e Snowflake

Aggiornare in parallelo `renderexpr_test.go` e `jsonexpr_test.go`; riusare `test/warehouseexpr/json.go` e `core/internal/datastore/query_json_test.go`:

- Stessa matrice del runtime: confronti S/N separati e membership S/N/B/Z, comprese liste solo Boolean, solo Null, false, String omonime, missing/SQL NULL/JSON Null e negazioni. Nessun `IN ()` con partizioni vuote.
- Number equivalenti con grafie diverse; soglie 76/37; coefficienti oltre 38 cifre e vicini positivi/negativi in Snowflake. String non genera ramo Number; Number non genera ramo String.
- Between omogenei, contains String/Number su JSON e array(json), array vuoti/misti e nessuna ricorsione negli array annidati.
- AST diretto in modalità tipizzata: kind, limiti e cardinalità verificati, Boolean/Null esclusivi della membership. Datastore imposta la modalità e conserva chiavi/valori raw.
- AST strutturale: stessa uguaglianza/negazione per tutti i kind; conservare gli snapshot Object e Number `1e76` di §8.6. Il nuovo controllo 76/37 non deve intercettare quel Number strutturale.
- Espressioni ordinarie identiche per lo stesso valore Go, anche con soglie int64 ricevute come Number API ai limiti signed/unsigned, float64 `1e100`, decimal, riferimenti Column, consensi e matching. Verificare literal interi esatti e nessuna modifica ai relativi renderer. Non introdurre COALESCE nei predicati ordinari né nuovi literal UTC nei datetime.
- Differenziali Go/SQL sui valori **riletti dallo storage**, con oracolo indipendente per i casi fondamentali. Separare le trasformazioni di storage Snowflake dai cambiamenti di semantica del filtro.
- Decimal ordinari Snowflake: verificare soglie oltre P,S della proprietà e limiti nativi del dialetto, senza applicare loro automaticamente l'algoritmo JSON né restringere core a P,S per aggirare limiti SQL.

Le suite richiedono l'infrastruttura già prevista. Non predisporla durante questa revisione; nell'implementazione la verifica sui due warehouse resta necessaria, oltre agli snapshot SQL.

### 13.5 Admin

Ampliare utility, `filters_parser_test.ts` e test UI/API:

- Tutti gli esempi §9.3; `true`/`false`/`null` diventano primitivi soltanto per membership JSON, quotati sempre String. Il cambio operatore richiede nuova validazione del contesto, senza conversioni silenziose del modello tipizzato.
- Campo vuoto distinto da `""` / `''`; apici/escape/backslash/spazi/newline e differenze fra controllo e linguaggio completo.
- Round-trip della lista `[100, "100", true, false, null, "true", "null", ""]`, anche con String contenenti spazi/virgolette/escape. Pruning e controlli di validità non eliminano false, Null o zero.
- Number nel JSON HTTP per tutti gli int/year, inclusi int64; `100.0` / `1e2` rifiutati prima di BigNumber. Verificare caricamento/modifica/salvataggio con `-9223372036854775808`, `9223372036854775807`, `18446744073709551615`, `Number.MAX_SAFE_INTEGER`, il suo successivo e `9007199254740993` (questi ultimi acquisiti come testo), più i casi fuori range. Nessun arrotondamento, String o wrapper nel payload; token interi senza frazione/esponente. Float64 `1e100` resta valido senza 76/37.
- Decimal/Number JSON: verifica 76/37 senza arrotondamento del coefficiente, corpus allineato a `types.IsDecimal`, zero/esponenti/zeri finali. Soglia decimal valida oltre P,S della proprietà.
- Body/query string esatti per JSON/decimal con `9007199254740993`, 76 cifre e 37 decimali; Number non quotato, String numerica quotata, Boolean/Null primitivi, nessun wrapper nel JSON.
- Decoder JSONbig: confronto differenziale sui confini dei 15 caratteri e sul ramo BigNumber lungo; non applicare 76/37 ai float ricevuti.
- Connessioni lista/dettaglio, griglia prima di aprire editor, caricamento/modifica/salvataggio/anteprima: stesso modello tipizzato e stesso payload normalizzato.
- Parser/serializer completo schema-aware e tutti i default dei connettori/Admin. Se lo schema arriva dopo il default, la conversione differita deve conservare provenienza e valore dei token.
- Datetime con offset: nessuna perdita dell'istante nel modello/trasporto; nessuna nuova normalizzazione funzionale.

Il test parser documenta `npx tsx`: verificarne l'inclusione nel processo di test, senza presumere che faccia parte della normale suite Admin. [Test parser](../../admin/src/utils/filters_parser_test.ts#L1).

### 13.6 Integrazione

Verificare pipeline nel **nuovo formato** su import da file/app, ingestion, estrazione utenti da eventi e dispatch; ricerca profili/eventi, listener, export con consensi/matching, create/update, ricaricamento stato e risposta API. Adeguare fixture/helper Go/TypeScript e costruttori diretti; nessuna conversione dei filtri precedenti fa parte di questo piano. Per il futuro rilascio resta soltanto l'avvertenza §11.

### 13.7 Verifiche della baseline già disponibili

Nell’analisi precedente sono stati eseguiti **test esistenti**, senza modificare sorgenti, test, fixture o configurazioni. La selezione ha coperto:

| Package | Test selezionati pertinenti | Risultato |
|---|---|---|
| `core` | Modello JSON, conversioni, validazione, limiti dei filtri, decimal e percorsi | Passati |
| `core/internal/state` | Lettura/scrittura `Where` e uguaglianza | Passati |
| `core/internal/filters` | `Test_Applies`, percorsi annidati e lettura proprietà | Passati |
| `warehouses/postgresql` | Rendering di espressioni, decimal JSON ed errori | Passati |
| `warehouses/snowflake` | Rendering di espressioni, decimal JSON ed errori | Passati |
| `tools/decimal` | `Test_Parse` | Passato |
| `tools/types` | `Test_IsDecimal`, `Test_Decode`, `Test_Marshal` | Passati |

Comando di selezione, eseguito con toolchain locale, dipendenze vendorizzate e cache temporanea:

```sh
go test ./core ./core/internal/state ./core/internal/filters \
  ./warehouses/postgresql ./warehouses/snowflake ./tools/decimal ./tools/types \
  -run '^(Test_Filter_.*|Test_convertFilter.*|Test_convertWhereToFilter|Test_parseFilterDecimal|Test_validate.*Filters|Test_validateFilter.*|Test_resolveFilterProperty|Test_unmarshalWhere.*|Test_Where.*Equal|Test_MarshalWhere.*|Test_Applies.*|Test_readPropertyFrom|Test_renderExpr.*|TestJSONDecimalRendering|TestJSONExpressionErrors|Test_Parse|Test_IsDecimal|Test_Decode|Test_Marshal)$' \
  -count=1
```

Sono stati esclusi `TestJSONExpressions` e `TestJSONFilters`, che richiedono infrastruttura warehouse. Non sono state eseguite le suite complete dei sette package. L'esperimento Node di §9.4 ha verificato soltanto le librerie esistenti; non è un test del nuovo editor. Questi risultati confermano le evidenze della baseline e la disponibilità di una parte delle verifiche per l'implementazione futura.

## 14. Piano di implementazione incrementale

Ogni intervento è una futura modifica; questa revisione non ne avvia alcuno. Le regole funzionali di §5 sono definitive. Il piano riguarda il nuovo formato e i suoi consumatori, senza convertitori, migrazioni o percorsi di compatibilità.

| Passo | Dipendenze | File/componenti | Intervento e criterio di completamento |
|---:|---|---|---|
| 1 — Decoder singolo operando | Contratto §3–5 | `state/where.go`, parser/utility esistenti, test pertinenti | Decoder specifico ispirato a `types.Decode`, con Number per tutti gli interi e sintassi/range esatti senza float64; year sintattico, float invariati, decimal/JSON Number 76/37 senza P,S; Boolean/Null solo membership. La PR sui tipi generali non è una dipendenza. Verificare disponibilità della correzione separata dello zero |
| 2 — Modello API e preparazione core | 1 | `core/filters.go`, `cmd`, `workspace.go`, `connection.go`, `pipeline.go`, `pipeline_validate.go`, test e `test/krenalistester` | `[]json.Value`, preparazione unica e risultato riusato dai chiamanti; adeguare costruttori per compilare. HTTP indipendente dall'ordine, errori e raccolta proprietà invariati |
| 3 — Persistenza e inversa | 1–2 | `state/where.go`, `load.go`/`keep.go`, `convertWhereToFilter`, test | Nuovo marshal/unmarshal e uguaglianza S/N/B/Z; tutti gli interi Number con serializer indipendente dalla bit size. Offset nella String API, politica persistita UTC e semantica SQL temporale conservate; nessun dual reader |
| 4 — Runtime JSON | 1–3 | `filters/applies.go`, `applies_test.go` | Confronti tipizzati e membership con presenza esplicita; matrici §5 vere anche per false/Null/assenza. Tipi ordinari e unari invariati |
| 5 — AST e renderer | 1–4 | `datastore/query.go`, `warehouses.go`, renderer/test entrambi i warehouse, `test/warehouseexpr` | Discriminante strutturale/tipizzato, nessun adattatore duale. SQL tipizzato S/N/B/Z e negazioni coerenti; snapshot strutturali di tutti i kind e SQL ordinario invariati; algoritmo numerico Snowflake preservato |
| 6 — Modelli e codec Admin | 1–2 | `api/types/pipeline.ts`, `utils/filters.ts`, `core/pipeline.ts`, dipendenza BigNumber/vendor | Draft/typed/transport distinti; parser gemello con contesto operatore, sintassi intera verificata prima della normalizzazione. Percorso FilterNumber→BigNumber→JSONbig per tutti gli interi, estremi 64 bit e round-trip esatti, anche oltre MAX_SAFE_INTEGER; float senza 76/37 |
| 7 — Editor, parser completo e HTTP | 6 e API pronta | `PipelineFilters`, `PipelineTransformation`, `useConnection`, `PipelinesGrid`, modelli Connection, `filters_parser.ts`, `api.ts`/`call.ts`, test | `Connections.find/get` normalizza i filtri; salvataggio e preview usano il risultato preparato. Parsing completo schema-aware, default conservati anche se differiti; nessun `JSON.stringify(filter)` né perdita di tipo |
| 8 — Chiusura integrazioni | 3–7 | File §10 e test §13 | Ricerca finale usi vecchi, rimozione duali/helper morti, regressioni sui percorsi attuali. Esecuzione delle verifiche appropriate, inclusi i due warehouse e Admin, documentando eventuali limiti |

I passi 2–5 condividono il contratto interno: spostare insieme le modifiche meccaniche necessarie a compilare, poi completare i casi semantici nel passo relativo. Non introdurre adattatori distribuiti per mantenere produttori e consumatori su due formati. I commit intermedi non sono unità di distribuzione indipendenti.

**Indispensabili:** decoder specifico, modello/preparazione, nuovo marshal/unmarshal, runtime, discriminante AST e due renderer, percorso Admin completo, chiamanti diretti e verifiche. **Opzionali:** compilazione/cache runtime, conservazione della grafia numerica originale e benchmark dopo una regressione misurata. Non servono nuovi operatori, nodi relazionali o schema fisico. Il problema dei dati precedenti rimane segnalato esclusivamente come avvertenza per il futuro rilascio (§11).

## 15. Decisioni ancora aperte e raccomandazioni

### 15.1 Decisioni funzionali definitive

Non restano decisioni funzionali aperte nel perimetro di questa revisione. A1–A6 sono approvate:

| ID | Decisione da rispettare |
|---|---|
| A1 | Between JSON con estremi entrambi String oppure entrambi Number |
| A2 | Membership JSON mista String/Number/Boolean/Null; Null presente distinto da assenza |
| A3 | Contains String: sottostringa o elemento String immediato; Number: solo elemento Number immediato |
| A4 | String JSON vuota esplicitamente quotata ammessa, distinta dal controllo vuoto |
| A5 | Compatibilità §5.2; Boolean/Null aggiunti esclusivamente alla membership |
| A6 | Tutti gli int/year come Number con sola sintassi intera, anche int64; frazioni/esponenti e String numeriche rifiutati |

Sono definitivi Number per tutti gli interi signed/unsigned e year, float con validazione corrente senza 76/37, decoder specifico delle soglie, nessun vincolo decimal P,S, nessuna progettazione di conversioni del formato precedente e preservazione della semantica SQL datetime. La divergenza temporanea dall'attuale formato String di int64 in `types.Decode`/`types.Marshal` è intenzionale; la futura PR sui tipi generali non blocca Typed Filter Values. L'Admin riconosce `true`/`false`/`null` non quotati soltanto nella membership JSON; quotati restano String. I nomi/firme tecniche esemplificati possono essere affinati nell'implementazione senza cambiare questi contratti.

### 15.2 Verifiche tecniche residue

| Verifica | Evidenza attuale | Criterio richiesto nell'implementazione |
|---|---|---|
| Discriminante nell'AST SQL | Raw `json.Value` attiva oggi uguaglianza strutturale, anche per Number `1e76` | Verificare tutti i costruttori e il default della modalità; nessuna regressione strutturale su scalari/Array/Object e nessuna validazione 76/37 anticipata su quel percorso |
| Membership Boolean/Null nei due SQL | Unari e guard di tipo esistono, la nuova membership non è implementata | Eseguire test differenziali per false, Null presente, missing, String omonime e negazioni, compresa la rigorosità di `AS_BOOLEAN` e partizioni vuote |
| Precisione Admin e confine HTTP | JSONbig conserva token lunghi; i brevi passano da Number; verifica mirata degli interi a 64 bit in §9.4 | Completare il round-trip Admin/HTTP ai limiti signed/unsigned e oltre MAX_SAFE_INTEGER, con sintassi intera verificata prima della normalizzazione e Number interi nel payload, senza esponenti. Conservare float fuori 76/37 e liste miste. Se emergono perdite, correggere il decoder degli operandi usando i token originali |
| Parser completo e schema disponibile | I default sono letti in `useConnection` prima del passaggio al modello pipeline | Verificare il punto preciso in cui lo schema è disponibile e conservare token/quoting fino alla conversione; controllare tutti i default senza reinterpretazioni implicite |
| Riuso del decoder raw | `json.Value.UnmarshalJSON` richiede receiver non inizializzato | Verificare unmarshal ripetuto, proprietà dei buffer e Null esplicito nel decoder dell'albero e standalone |
| Zero ed esponente estremo | La baseline precede la correzione autonoma `9c45ac229` | Verificare disponibilità della correzione e test di sintassi/range; nessuna eccezione duplicata nei filtri |
| Decimal ordinari Snowflake | [Serializer](../../warehouses/snowflake/renderexpr.go#L554) usa literal diretti, distinto dal ramo JSON | Verificare i limiti effettivi del dialetto con soglie oltre P,S, senza promettere 76 cifre native né restringere core a P,S |
| Round-trip datetime | Inversa API omette offset; persistenza UTC e renderer senza normalizzazione sono già implementati | Conservare l'istante nella String tipizzata e verificare separatamente API, persistenza e SQL, lasciando invariata la semantica di ciascun percorso |

La prima implementazione può usare il parsing runtime semplice e i meccanismi numerici SQL già presenti. La precisione va protetta lungo tutto il percorso, compresi liste miste, caricamento API, griglia/editor, anteprime, persistenza e costruttori Go.

La modifica è fattibile senza alterare l'architettura generale dei filtri. Richiede interventi coordinati sui consumatori e il discriminante SQL per preservare l'uso strutturale interno; non richiede una nuova infrastruttura di schema o query. Restano verifiche tecniche sul codice futuro e sui due warehouse, non approvazioni funzionali delle regole già definite. Nessuna implementazione è avviata da questo documento.

La rappresentazione intera è unificata e non richiede condizioni sulla bit size nei serializer né modifiche ai renderer degli interi. La specifica è coerente e pronta per l'implementazione di Typed Filter Values; l'adeguamento futuro di `types.Decode`/`types.Marshal` è una PR indipendente, senza dipendenze bloccanti per questa attività.
