# Audit dei filtri attuali e delle integrazioni previste

**Data:** 25 ottobre 2026. **Baseline eseguibile:** branch `typed-filter-values`, commit `e66eea846418c592b27689eff2c4ba4c06fac1b9`. **Destinazione:** preparazione della Fase 4B del behavioral filtering.

Questo audit aggiorna [09-audit-filtering-legacy-vincoli-implementazione.md](09-audit-filtering-legacy-vincoli-implementazione.md). Il documento 09 rimane la fotografia precedente. La specifica di riferimento per il significato è [06-specifica-semantica-normativa-congelata-v1.0.md](06-specifica-semantica-normativa-congelata-v1.0.md); la rappresentazione tecnica è definita da [07-ast-canonico-progetto-tecnico-completo.md](07-ast-canonico-progetto-tecnico-completo.md), con riepilogo in [08-ast-canonico-modello-finale-estratto.md](08-ast-canonico-modello-finale-estratto.md). Questi tre testi sono disponibili e sono stati confrontati con il codice.

## 1. Valutazione complessiva

La base attuale è più completa di quella descritta nel documento 09. I filtri accettano operandi JSON tipizzati; il supporto JSON comprende confronti, percorsi annidati, contenimento, membership mista, null, vuoto ed esistenza; sono disponibili test differenziali fra valutatore Go e warehouse. L'Admin distingue il modello dell'editor, il filtro tipizzato e il trasporto HTTP. Le precedenti conclusioni su JSON non compilabile, operandi sempre testuali, temporal operator ammessi su JSON e assenza di un modello editoriale separato sono superate.

Il sistema rimane un filtro su proprietà di un singolo record. Non implementa ancora le cinque primitive comportamentali, il registro semantico della Fase 4A o la valutazione a tre risultati. La tipizzazione degli operandi rende più solida la base, ma non risolve da sola la traduzione semantica al nuovo linguaggio.

Per la Fase 4B restano determinanti quattro questioni:

1. **Preservazione del significato:** il legacy restituisce booleani e alcuni operatori negativi includono proprietà assenti; il target prescrive `UNKNOWN` per confronti ordinari su dati ignoti.
2. **Rappresentabilità:** il legacy ammette quattro livelli di gruppi, il target tre. La compatibilità non può essere dedotta dal solo mapping delle foglie.
3. **Dati persistiti:** `typed-filter-values` cambia il formato degli operandi senza un convertitore dei precedenti filtri JSON. Il trattamento prima del rilascio è già richiesto da `RELEASE-UPGRADE.md`.
4. **Contratti semantici:** sono disponibili tipi e semantiche di valore; sono previsti a breve ruoli delle proprietà del profilo. Identità delle azioni, grana, repertori, completezza e contratti di evidenza richiedono ulteriori componenti.

Il percorso verso la Fase 4B è praticabile riusando validazione, tipi e confini esistenti. Richiede un modello semantico distinto da `core.Filter`, `state.Where` e `warehouses.Expr`.

## 2. Revisioni esaminate e stato delle integrazioni

| Riferimento locale | Revisione esaminata | Ruolo nell'audit |
|---|---|---|
| `typed-filter-values` | `e66eea846418c592b27689eff2c4ba4c06fac1b9` | Stato attuale; sorgenti letti dal checkout e test eseguiti. |
| `json-filter-support` | `82e7be3a1939340cfc3374cf9f8458a63f21a43b` | Già incluso nella storia della baseline; aiuta a distinguere completamento JSON e successiva tipizzazione. |
| `profile-roles` | `f4f9101f1999c3b4c46c77d90a122af6001a6cb2` | Branch locale consultato. |
| `origin/profile-roles` | `0014b6843086d2ac9f3dfd574f71377627868619` | Riferimento disponibile localmente più recente; fonte principale dell'approfondimento sui ruoli. |
| `profiles-list-redesign` | `95f45859515df3b727cb712985efd13ec78e697b` | Branch locale consultato. |
| `origin/profiles-list-redesign` | `2f0a2fb35c9d7459f56cbaa6186431f4a8a394f1` | Riferimento disponibile localmente più recente; fonte principale dell'approfondimento sulla pagina Profiles. |

I riferimenti `origin/...` sono quelli presenti nel repository al momento dell'analisi, senza fetch. Non è stato cambiato branch né prodotto un merge di prova. I sorgenti dei branch aggiuntivi sono stati letti con `git show`; i test riportati in §18 riguardano la baseline `typed-filter-values`.

Secondo il piano comunicato dal responsabile del progetto, `profile-roles` è in revisione finale e verrà incluso a breve; `profiles-list-redesign` è in revisione iniziale e verrà anch'esso incluso, aggiornandolo al sistema di filtri di `typed-filter-values`. Il formato precedente presente nel redesign è quindi una base da adeguare durante l'integrazione. Le sue funzionalità di prodotto sono rilevanti per la Fase 4B, ma non sono già presenti nel checkout attuale.

I riferimenti al codice senza indicazione di branch riguardano `e66eea846`. Per i branch aggiuntivi, file e linee sono indicati insieme al riferimento della tabella, in modo da non attribuire al checkout codice che non contiene.

## 3. Che cosa cambia rispetto al documento 09

| Conclusione precedente | Stato aggiornato |
|---|---|
| `FilterCondition.Values` è `[]string`. | È `[]json.Value`; il kind dell'operando è parte della richiesta. |
| Validazione e conversione sono due passaggi con parsing ripetuto. | `prepareFilter` valida e costruisce `state.Where` nella stessa visita. |
| Il limite ordinario degli operandi testuali è 60 rune. | È 80 rune; rimangono eccezioni per valori stringa ammessi e per String JSON vuota. |
| Confronti JSON possono arrivare al panic su `JSONConditionValue`. | Il tipo duale è stato eliminato; runtime e renderer ricevono operandi JSON tipizzati. |
| Solo alcuni test booleani su path JSON sono compilabili. | `BaseExpr.Keys` rappresenta i segmenti del percorso; il ramo JSON copre gli operatori pubblici ammessi. |
| Il backend permette operatori temporali su JSON. | Sono rifiutati; rimangono riservati a datetime, date, time e year. |
| `is empty` su object può superare il validator e fallire nel compiler. | Object è escluso dall'operatore; rimangono string, JSON, array e map. |
| Null ed esistenza JSON sono confusi nel SQL. | Il ramo JSON distingue JSON Null presente e percorso assente; i due operatori rimangono distinti nel compiler. |
| Non esiste una matrice differenziale fra valutatore e warehouse. | Esiste per JSON e relativi contenitori, con fixture condivise e controllo della membership. |
| L'Admin modifica il medesimo tipo usato per l'API. | Sono separati `EditorFilter`, `Filter` e `TransportFilter`. |
| Mancano semantiche di valuta e unità nei tipi. | `tools/types.Type` contiene già semantiche e opzioni per denaro, valuta, misure e durata. |
| Profiles nell'Admin invia un filtro nullo. | Ancora vero nella baseline; sarà superato dall'integrazione del redesign già in revisione. |
| Frozen e Fase 4A non sono accessibili e il confronto è `UNVERIFIED`. | I testi sono disponibili; le differenze di significato e struttura sono ora confrontabili direttamente. |

Evidenze: [modello pubblico](../core/filters.go#L97), [preparazione](../core/filters.go#L502), [decoder degli operandi](../core/internal/state/filter_operand.go#L25), [compiler](../core/internal/datastore/query.go#L58), [tipi Admin](../admin/src/lib/api/types/pipeline.ts#L93), [semantiche dei tipi](../tools/types/semantics.go#L14), [suite differenziale](../core/internal/datastore/query_json_test.go#L33).

## 4. Architettura ed esecuzione attuali

```text
JSON HTTP
  → cmd: decodifica strutturale in core.Filter
  → core.prepareFilter: schema, operatori, cardinalità, valori e limiti
  → state.Where con operandi Go già decodificati
      → filters.Applies per un record in memoria
      → datastore.convertWhere → warehouses.Expr → PostgreSQL o Snowflake
      → state.MarshalWhere → pipelines.filter e notifiche di stato
          → unmarshalWhere con InSchema → conversione inversa all'API
```

`cmd` non ha bisogno dello schema per leggere Number, String, Boolean e Null. Lo schema viene risolto in `core`, che controlla compatibilità e costruisce l'espressione interna. I chiamanti ricevono anche i path base usati: una condizione su `properties.amount` richiede il campo JSON `properties`, senza trasformare il path interno in una nuova colonna di input. Le restrizioni su quali pipeline ammettono filtri rimangono nel controllo del pipeline; `prepareFilter` non riceve più ruolo e target.

Il percorso SQL è usato dalle query Profiles/Events e dai percorsi warehouse; il valutatore Go opera fra l'altro negli import, nei listener e nell'acquisizione degli eventi. Questa doppia esecuzione resta rilevante: l'allineamento JSON non dimostra automaticamente quello di tutti i tipi ordinari.

Riferimenti: [Profiles HTTP](../cmd/workspaces.go#L882), [Profiles core](../core/workspace.go#L1251), [validazione pipeline](../core/pipeline_validate.go#L154), [ApplicationUsers](../core/connection.go#L242), [import](../core/run_import.go#L178), [collector](../core/internal/collector/collector.go#L850), [observer](../core/internal/collector/observer.go#L177).

La root attuale riguarda le proprietà del record interrogato. Un filtro nell'endpoint Events seleziona eventi; non seleziona automaticamente profili che abbiano una certa storia di eventi. Un conteggio di profili risultanti non è una riduzione comportamentale all'interno di una condizione.

## 5. Grammatica pubblica e limiti

La struttura rimane `Filter {operator, rules}` con foglie `FilterCondition {property, operator, values}`. La root è un gruppo AND/OR; non esiste NOT generico. Sono validi i singleton; i gruppi vuoti sono invalidi. Il filtro assente rappresenta l'assenza di vincoli; gli endpoint Profiles ed Events rifiutano invece il parametro esplicito `filter=null`.

La profondità massima è quattro gruppi, root inclusa. Le cento regole massime comprendono condizioni e gruppi discendenti, escludendo la root. Il decoder dell'albero controlla struttura, proprietà JSON sconosciute/duplicate e limiti; la preparazione verifica anche gli alberi costruiti direttamente in Go. Non viene eseguita una normalizzazione generale di ordine o grouping.

Gli otto operatori senza operandi richiedono `Values == nil`: omissione e `values:null` rappresentano tale stato; `values:[]` viene rifiutato. Un elemento di lista `null` è adesso un vero operando Null, valido soltanto dove autorizzato.

I path pubblici sono nomi di proprietà separati da punti, secondo `types.IsValidPropertyPath`. Non sono una sintassi di JSONPath: non offrono indici di array, wildcard, navigazione automatica degli elementi o chiavi arbitrarie citate. I test del renderer con chiavi letterali contenenti punti o apici esercitano la rappresentazione interna a segmenti, più ampia della sintassi pubblica.

Riferimenti: [limiti e modello](../core/filters.go#L24), [decoder ricorsivo](../core/filters.go#L371), [gruppi](../core/filters.go#L722), [cardinalità](../core/filters.go#L659), [path](../tools/types/properties.go#L287).

## 6. Operandi tipizzati e catalogo degli operatori

### Rappresentazione dei valori

| Tipo della proprietà o elemento ammesso | Operando pubblico | Valore interno |
|---|---|---|
| String, UUID, IP | String | String; UUID/IP normalizzati. |
| Int signed/unsigned, anche a 64 bit | Number con sintassi intera | `int`/`uint`, parsing esatto entro i rispettivi limiti a 64 bit. |
| Year | Number con sintassi intera | Intero fra 1 e 9999. |
| Float | Number | `float64`, validato alla bit size della proprietà. |
| Decimal | Number | `decimal.Decimal`, entro precisione 76 e scala 37. |
| DateTime, Date, Time | String temporale | `time.Time`. |
| JSON e percorso interno JSON | String/Number; anche Boolean/Null per membership | `json.Value` che conserva il kind. |
| Boolean, Object, Map | Nessun nuovo operando scalare | Solo gli operatori senza valori ammessi per quel tipo. |

Gli operandi sono soglie, non nuovi dati da inserire nella colonna: un confronto con una proprietà `decimal(5,2)` può usare una soglia più ampia, nei limiti globali 76/37. Il limite 76/37 non si applica ai float; gli interi non ereditano il range più stretto della bit size della colonna. Per gli interi `100.0`, `1e2` e `"100"` sono rifiutati, mentre JSON Number 100, 100.0 e 1e2 sono equivalenti nei confronti su proprietà JSON.

Le stringhe ordinarie devono essere non vuote, UTF-8 valide, senza NUL e di massimo 80 rune. I valori esplicitamente ammessi da uno schema stringa seguono il relativo elenco. La String vuota è inoltre ammessa per JSON, dove va distinta da un controllo editoriale ancora vuoto.

Riferimenti: [DecodeFilterOperand](../core/internal/state/filter_operand.go#L25), [controlli schema e valori](../core/filters.go#L519), [limiti testuali](../core/internal/util/util.go#L79), [regressioni numeriche](../core/internal/state/filter_operand_test.go#L51).

### Catalogo completo

Le righe raggruppano operatori con la stessa applicabilità; il catalogo comprende tutti i 26 operatori pubblici. S/N/B/Z indicano String, Number, Boolean e Null dell'operando JSON.

| Operatori | Tipi ammessi e restrizioni | Cardinalità |
|---|---|---:|
| `is`, `is not` | Scalari escluso Boolean; JSON soltanto S/N. | 1 |
| `is less than`, `is less than or equal to`, `is greater than`, `is greater than or equal to` | Int, Float, Decimal, String senza elenco ammesso, JSON S/N. | 1 |
| `is between`, `is not between` | Tipi ordinabili previsti dal validator, inclusi temporali/year; esclusi Boolean, UUID/IP e contenitori; String senza elenco; JSON con due S o due N. | 2 |
| `contains`, `does not contain` | String, JSON, array di elementi che supportano `is`; JSON S/N. | 1 |
| `is one of`, `is not one of` | Int, Float, Decimal, temporali/year, String, JSON; esclusi Boolean ordinario, UUID/IP e contenitori. JSON ammette liste miste S/N/B/Z. | Almeno 1 |
| `starts with`, `ends with` | String senza elenco ammesso; JSON soltanto S. | 1 |
| `is before`, `is on or before`, `is after`, `is on or after` | DateTime, Date, Time, Year. JSON escluso. | 1 |
| `is true`, `is false` | Boolean e JSON. | 0 |
| `is empty`, `is not empty` | String, JSON, Array, Map; Object escluso. Un elenco stringa deve ammettere il vuoto. | 0 |
| `is null`, `is not null` | Proprietà nullable oppure JSON. | 0 |
| `exists`, `does not exist` | Proprietà read-optional oppure percorso interno JSON. | 0 |

Per `contains` su `array(json)` restano ammessi solo operandi String/Number. Le liste membership non vengono ordinate o deduplicate. `between` mantiene l'ordine ricevuto: non inverte estremi rovesciati. La matrice è definita nel [validator core](../core/filters.go#L536) e riprodotta nell'[Admin](../admin/src/lib/core/pipeline.ts#L66).

## 7. Semantica JSON attuale e differenze residue fra executor

### Confronti e negazioni JSON

Su JSON, il kind dell'operando decide il dominio del confronto. Number 100 non è String `"100"`; `is greater than [100]` non confronta stringhe numeriche. Gli intervalli richiedono estremi dello stesso kind. `contains` con String cerca una sottostringa oppure un elemento String immediato; con Number cerca un elemento Number immediato. Non cerca ricorsivamente in oggetti o array annidati.

La membership ammette Boolean e Null oltre a String e Number. `[null]` corrisponde a JSON Null presente; non corrisponde a un path assente. `is false` controlla il falso booleano: non è la negazione di `is true`.

I negativi JSON sono complementi booleani del predicato positivo. I renderer applicano il tipo corretto, trasformano l'esito SQL indeterminato in false e applicano all'esterno l'eventuale negazione. Per un normale campo JSON rappresentato nel warehouse:

| Stato | `is [100]` | `is not [100]` | `is one of [null]` | `is null` | `is not null` | `exists` | `is empty` |
|---|---|---|---|---|---|---|---|
| Assente o estrazione SQL NULL | false | true | false | false | true | false | true |
| JSON Null presente | false | true | true | true | false | true | true |
| Number 100 | true | false | false | false | true | true | false |
| String `"100"` | false | true | false | false | true | true | false |
| String vuota, array vuoto o oggetto vuoto | false | true | false | false | true | true | true |

La tabella descrive il ramo JSON, non tutte le colonne SQL. Il runtime Go distingue inoltre la presenza della chiave dal suo valore: un `nil` esplicitamente presente in una mappa può essere trattato come null. Nello storage dei profili SQL NULL rappresenta invece assenza, mentre JSON Null rimane un valore presente; il confronto differenziale deve usare la rappresentazione effettivamente conservata.

Evidenze: [dispatch e presenza](../core/internal/filters/applies.go#L45), [confronto JSON](../core/internal/filters/applies.go#L109), [membership](../core/internal/filters/applies.go#L291), [null e vuoto](../core/internal/filters/applies.go#L511), [PostgreSQL](../warehouses/postgresql/renderexpr.go#L269), [Snowflake](../warehouses/snowflake/renderexpr.go#L255).

### Tipi ordinari e oggetti appiattiti

L'allineamento JSON non elimina le differenze legacy dei tipi ordinari. In Go, `is not`, `is not between`, `does not contain` e `is not one of` negano il risultato booleano positivo. Su un valore mancante possono quindi essere true. I renderer ordinari usano invece `<>`, `NOT BETWEEN`, `NOT IN` e confronti SQL senza la trasformazione prevista dal ramo JSON: su SQL NULL non selezionano la riga.

Anche `is null`/`exists`, dove ammessi, hanno confini diversi fra mappe e colonne ordinarie: la mappa conserva la presenza separata dal null; una colonna SQL non la conserva automaticamente. Per i profili, le proprietà dello schema sono non-nullable ma le colonne fisiche sono nullable, così che NULL rappresenti l'assenza del valore.

Un object di schema viene appiattito in colonne discendenti. `exists` diventa OR delle presenze dei discendenti; `does not exist` diventa AND delle assenze. Non c'è un bit autonomo che distingua un object originariamente presente ma vuoto da uno assente. Questo è un limite della rappresentazione, non il precedente difetto del compiler JSON.

Riferimenti: [renderer PostgreSQL ordinario](../warehouses/postgresql/renderexpr.go#L84), [renderer Snowflake ordinario](../warehouses/snowflake/renderexpr.go#L94), [object nel compiler](../core/internal/datastore/query.go#L124), [colonne del profilo](../core/internal/datastore/utils.go#L165).

## 8. Compiler SQL e limiti di rappresentazione

`datastore.convertWhere` conserva gli operatori, copia gli operandi e imposta `JSONTypedComparison` per JSON e array JSON. I segmenti successivi alla colonna JSON sono conservati in `BaseExpr.Keys`. I renderer distinguono questo uso dall'uguaglianza JSON strutturale dei chiamanti interni, che mantiene il modo predefinito `JSONStructuralComparison`. La presenza di `json.Value` non basta quindi da sola a scegliere la semantica del confronto.

Il ramo JSON verifica modo, operatore, numero degli operandi, JSON valido, kind compatibile e limiti decimal. Non si basa più sul duale String/Number. I negativi non richiedono un NOT pubblico nell'albero: è il renderer a implementare il complemento del singolo operatore.

Per PostgreSQL, i numeri JSON vengono confrontati come `numeric` e il testo usa collazione `C`. Per Snowflake il codice usa la rappresentazione `TO_JSON` del valore conservato e `DECFLOAT`; gestisce gli operandi con oltre 38 cifre significative adattando il bordo dei confronti ordinati ed escludendoli dall'uguaglianza numerica quando non rappresentabili dal dato conservato. Il ramo String rimane distinto. I test differenziali rileggono i dati dal warehouse prima di calcolare l'atteso in Go, così da confrontare la stessa rappresentazione.

Queste garanzie non recuperano informazione persa prima della query. Arrotondamenti nello storage Snowflake, trattamento di NUL nelle stringhe JSON e appiattimento degli object mantengono i limiti descritti nel [README del datastore](../core/internal/datastore/README.md#L82). Non è corretto attribuirli a un'ulteriore restrizione del linguaggio degli operatori JSON.

Riferimenti: [BaseExpr e modo JSON](../warehouses/warehouses.go#L824), [compilazione](../core/internal/datastore/query.go#L100), [confronto PostgreSQL](../warehouses/postgresql/renderexpr.go#L249), [confronto Snowflake](../warehouses/snowflake/renderexpr.go#L557), [rilettura e verifica membership](../core/internal/datastore/query_json_test.go#L220).

### Tempi

Il round-trip pubblico dei datetime conserva l'offset mediante RFC3339Nano; la persistenza `Where` scrive e rilegge l'istante in UTC. Date e Time hanno formatter specifici, anche per gli operandi degli array. I problemi di serializzazione interna indicati nel vecchio audit non possono essere trasferiti automaticamente al nuovo formato.

Rimane però distinto il rendering SQL ordinario dei datetime: i test mantengono literal senza offset, con precisione specifica del warehouse. Per esempio l'operando `12:00 +02:00` viene reso con ora `12:00` nei literal SQL di regressione. Questo comportamento non costituisce la normalizzazione coerente degli istanti richiesta dal futuro modello temporale; andrà considerato nella traduzione, senza confonderlo con il round-trip API/persistenza corretto.

Evidenze: [conversione inversa](../core/filters.go#L200), [persistenza temporale](../core/internal/state/where.go#L312), [test offset](../core/filters_typed_test.go#L22), [regressione PostgreSQL](../warehouses/postgresql/renderexpr_typed_test.go#L57), [regressione Snowflake](../warehouses/snowflake/renderexpr_typed_test.go#L57).

## 9. Persistenza e compatibilità di rilascio

I filtri pipeline rimangono in `pipelines.filter`, colonna JSONB del database Krenalis. `state.Where` usa path a segmenti e operatori interni, quindi non coincide con il JSON pubblico. Il caricamento usa `InSchema`; lo stesso formato transita nelle notifiche di stato. Le modifiche coinvolgono pertanto database, processi che ricaricano lo stato e risposte API.

Esempio del formato attuale:

```json
{
  "operator": "And",
  "rules": [
    {"property": ["account"], "operator": "Is", "values": [9223372036854775807]},
    {"property": ["properties", "status"], "operator": "IsOneOf", "values": [100, "100", true, null]}
  ]
}
```

`MarshalWhere` conserva Number, String, Boolean e Null; il loader usa `DecodeFilterOperand`, controlla cardinalità e kind e ricostruisce i tipi senza passare gli interi o i decimal attraverso float64. I confronti di uguaglianza JSON di `WhereCondition.Equal` sono semantici per kind: Number 100 e 1e2 sono uguali, Number 100 e String `"100"` no. L'albero rimane comunque ordinato e strutturale.

Prima della tipizzazione un operando JSON testuale poteva essere interpretato sia come testo sia come numero, a seconda del dato. Nel nuovo formato una String numerica indica soltanto testo. Non esiste quindi una conversione universale che consista nel cambiare semplicemente il tipo Go o nel provare a parsare ogni stringa come numero: occorre stabilire il significato voluto per i filtri reali. Alcuni vecchi payload possono essere rifiutati; altri possono restare leggibili cambiando membership.

Il commit non introduce un lettore legacy, un campo versione o un updater di compatibilità. [RELEASE-UPGRADE.md](../core/internal/initdb/RELEASE-UPGRADE.md#L8) registra esplicitamente il lavoro necessario su `pipelines.filter` e notifiche prima del rilascio. La migrazione al formato tipizzato e la futura traduzione al Canonical AST sono due passaggi distinti.

Riferimenti: [schema SQL](../core/internal/initdb/schema.sql#L203), [serializer](../core/internal/state/where.go#L249), [loader](../core/internal/state/where.go#L357), [caricamento pipeline](../core/internal/state/load.go#L617), [notifiche](../core/internal/state/keep.go#L631), [scrittura pipeline](../core/pipeline.go#L836), [test JSONB](../core/internal/state/filter_operand_test.go#L176).

## 10. Editor Admin e authoring attuali

L'Admin distingue ora tre rappresentazioni:

| Modello | Responsabilità |
|---|---|
| `EditorFilter` | Controlli testuali incompleti; operatore ancora vuoto; `valueKinds` per conservare l'intenzione durante le modifiche. |
| `Filter` | Valori String, Boolean, Null o `FilterNumber {kind, text}`; il testo numerico evita una conversione anticipata in Number JavaScript. |
| `TransportFilter` | Number rappresentati da BigNumber per l'encoder JSON; nessun wrapper numerico nel payload HTTP. |

`prepareEditorFilter` elimina le righe senza proprietà, controlla schema, limiti, operatori e valori, e produce il filtro tipizzato. Non scarta indistintamente ogni condizione incompleta: una condizione con proprietà scelta e operatore o valore mancante può produrre un errore. `valueKinds` impedisce di reinterpretare silenziosamente un valore dopo un cambio di proprietà o operatore.

`toEditorFilter` mantiene le virgolette necessarie a distinguere String e Number. Il parser testuale accetta lo schema; `parseJSONFilterText` interpreta i letterali Boolean/Null solo per membership. `encodeFilter` usa il medesimo percorso numerico esatto per i filtri nelle query string; i body passano attraverso `toTransportFilter` e l'encoder HTTP. Le risposte vengono adattate da `decodeFilter`.

La catena delle dipendenze è verificabile nel codice vendorizzato: il caller costruisce BigNumber dal testo, `JSONbig.stringify` riconosce BigNumber prima di `toJSON` e ne emette il risultato senza virgolette. Il parser `json-bigint` usa il testo originale per i numeri lunghi. Riferimenti: [adattatori](../admin/src/utils/filters.ts#L410), [trasporto](../admin/src/utils/filters.ts#L469), [HTTP](../admin/src/lib/api/call.ts#L6), [stringify vendorizzato](../admin/node_modules_vendor/json-bigint/lib/stringify.js#L218), [parse vendorizzato](../admin/node_modules_vendor/json-bigint/lib/parse.js#L205).

Il builder corrente è ancora `PipelineFilters`; i path posizionali forniscono gli identificatori delle righe, senza un'identità persistente del nodo. Ordine e grouping restano parte della struttura modificabile. La pagina Profiles della baseline continua a chiamare `find(..., null, ...)`: l'adozione del filtro da parte del listing arriva con il redesign descritto in §13.

Riferimenti: [tipi](../admin/src/lib/api/types/pipeline.ts#L93), [preparazione editor](../admin/src/lib/core/pipeline.ts#L2045), [validazione valori](../admin/src/lib/core/pipeline.ts#L2080), [parser](../admin/src/utils/filters_parser.ts#L244), [builder](../admin/src/components/routes/PipelineWrapper/PipelineFilters.tsx#L65), [listing attuale](../admin/src/components/routes/Profiles/useProfiles.ts#L121).

## 11. Schema dei profili e registro semantico

È necessario distinguere tre livelli già presenti o previsti:

1. **Tipo del valore:** nella baseline esistono già semantiche country, email, phone, URL, money, percentage, measurement e duration. Money può avere una valuta, Measurement un'unità, Duration la relativa unità. Queste informazioni sono riutilizzabili.
2. **Ruolo della proprietà corrente:** `profile-roles` associa un significato di profilo a un path concreto, indipendentemente dal suo nome fisico. È un'integrazione prevista a breve.
3. **Registro della Fase 4A:** deve descrivere reference e revisioni del significato, ruoli degli eventi, grana, identità, repertori, contratti autorevoli ed evidenze certificate. I primi due livelli non ne forniscono automaticamente tutte le garanzie.

Non è quindi corretto ripetere che lo schema attuale descrive soltanto storage e non contiene valuta/unità. È invece corretto dire che non contiene ancora il registro completo necessario al behavioral filtering. La sola presenza di `money` non certifica che un campo sia importo pagato per ordine; il ruolo Email non certifica contattabilità o consenso.

Evidenze nella baseline: [Property e Type](../tools/types/types.go#L141), [catalogo semantiche](../tools/types/semantics.go#L14), [Currency, DurationUnit e UnitOfMeasure](../tools/types/semantics.go#L299), [workspace e fonti primarie](../core/internal/state/state.go#L908). Requisiti target: documento 06 §§9–10, 15–16 e 22; documento 08 F2, F5 e F6.

## 12. Apporto previsto del branch profile roles

Nel riferimento `origin/profile-roles@0014b6843`, `ProfileRoleAssignments` contiene `firstName`, `lastName`, `email`, `country` e `photo`. Ogni ruolo punta al path di una proprietà; il path vuoto indica nessuna assegnazione. Il server valida esistenza, sintassi, unicità dell'assegnazione e compatibilità con tipo/semantica. Per esempio Email richiede semantica email, Country country e Photo URL; nome e cognome richiedono String senza altra semantica.

Le assegnazioni sono dati del workspace e vengono applicate con lo schema nel processo di alterazione, con aggiornamento del database e dello stato. Il branch include controlli per gli aggiornamenti rifiutati e test di persistenza/coerenza fra ruoli e schema. L'Admin le usa nella configurazione e nella presentazione del profilo. Il riferimento più recente aggiunge anche l'assegnazione alla creazione del workspace.

Fonti del branch: `core/workspace.go:55`, `core/alter_profileschema.go:292`, `core/core.go:1727` nell'applicazione finale dello schema, `core/organization.go:530` per la creazione, `core/internal/state/load.go`, `core/internal/state/keep.go`, `core/profile_roles_test.go:15`, `test/profile_role_assignments_test.go:21`, `admin/src/components/helpers/profileRoles.ts`. Riproduzione della lettura: `git show 0014b6843086d2ac9f3dfd574f71377627868619:core/workspace.go` e analogamente per gli altri file.

L'apporto è pertinente e va riusato per le proprietà correnti. Rispetto alla Fase 4A rimangono da definire ruoli di azioni come Purchase/Refund, identità Order/Product/Message, namespace e deduplicazione, grana, revisione immutabile del significato e contratti di completezza. Le assegnazioni attuali sono path modificabili del workspace: non costituiscono già un protocollo per mantenere invariato il significato dei filtri salvati dopo una riassegnazione.

**Conseguenza per la Fase 4B:** progettare il collegamento fra questi ruoli e i reference del filtro; evitare un secondo sistema scollegato di assegnazione delle stesse proprietà. Identità del profilo, ruolo di una proprietà e identità di un oggetto business devono comunque restare concetti distinti. L'inclusione del branch non richiede di considerare già implementato il registro comportamentale.

## 13. Apporto previsto del redesign della pagina Profiles

Il riferimento `origin/profiles-list-redesign@2f0a2fb35` modifica direttamente l'authoring e l'esecuzione dei filtri nel listing. La sua storia include già lavoro sui ruoli del profilo, ma non l'intera revisione corrente di `origin/profile-roles`; l'integrazione dovrà conservare lo stato approvato dei due filoni.

### Funzionalità pertinenti

Il branch estrae un `FilterEditor` condiviso da pipeline e Profiles. `ProfilesFilters` distingue bozza, filtro applicato e candidato in anteprima; offre riepilogo, conteggio e cronologia della sessione. La cronologia ha un limite di 50 voci e non equivale a un catalogo persistito di segmenti. La lista invia il filtro applicato nelle richieste, superando il filtro sempre nullo della baseline.

Il backend aggiunge `Workspace.ProfileCount` e l'endpoint `/profiles/count`. Listing e conteggio ricevono lo schema pertinente, verificano l'allineamento delle dipendenze con quello corrente e restituiscono errori specifici di disallineamento. Il listing aggiunge `hasNext`, limiti espliciti di paginazione e un ordinamento con KPID come spareggio. La UI usa identificativi di esecuzione e cancellazione per evitare che risposte superate sostituiscano lo stato corrente.

Queste funzionalità sono punti d'integrazione utili per il futuro modello. `profilesSequence` riguarda la sequenza di pagine/richieste del listing: non implementa percorsi comportamentali P. Il conteggio di anteprima misura i profili selezionati; non è una spiegazione delle evidenze né distingue automaticamente esclusioni certe e `UNKNOWN`.

Gli identificativi di esecuzione e l'allineamento dello schema proteggono la coerenza della sessione UI, senza certificare uno snapshot dei dati comune a tutte le richieste. Nei renderer Query del branch, conteggio e lettura sono chiamate distinte; anteprima e applicazione sono a loro volta richieste separate. Il requisito target di una valutazione a un unico T richiede un proprio contratto di esecuzione, da integrare con questa esperienza UI. Fonti del branch: `warehouses/postgresql/query.go:38` e `warehouses/snowflake/query.go`, oltre al documento 06 §3.

Fonti del branch: `admin/src/components/base/FilterEditor/FilterEditor.tsx:94`, `admin/src/components/routes/Profiles/ProfilesFilters.tsx:31`, `profilesFilterSession.ts:5`, `useProfiles.ts:428`, `profilesSequence.ts:80`; `core/workspace.go:1228` e `:1307`; `core/internal/datastore/store.go:528` e `:578`; `cmd/workspaces.go:919`; `test/profiles_pagination_test.go:23`, `admin/tests/profiles.spec.ts` e `admin/tests/profiles-sequence.spec.ts`. Questi riferimenti appartengono al commit `2f0a2fb35c9d7459f56cbaa6186431f4a8a394f1`.

### Adeguamento già previsto al sistema di filtri attuale

Il redesign usa la precedente base main: valori stringa, validazione/conversione separate e serializzazione dei filtri con `JSON.stringify`. Il piano comunicato prevede di aggiornarlo al sistema di questo branch. I seguenti sono quindi criteri per quell'adeguamento e per la verifica del risultato integrato:

- Portare `FilterEditor` e `ProfilesFilters` sulla separazione `EditorFilter` → `Filter` → `TransportFilter`, conservando kind, precisione e virgolette durante modifica, annulla/ripristina, anteprima e applicazione.
- Usare il codec dei filtri anche nel nuovo endpoint di conteggio; allineare i nuovi chiamanti core a `prepareFilter`, mantenendo controllo delle dipendenze e allineamento dello schema.
- Conservare `JSONTypedComparison` nel compiler e la semantica JSON completa: il merge del redesign non deve ripristinare i rami precedenti.
- Rendere coerenti i fingerprint delle sessioni e delle query con il modello scelto. Oggi il redesign ordina le chiavi degli oggetti, conserva l'ordine degli array e normalizza l'operatore dei singleton per un fingerprint dei risultati: questo non è l'uguaglianza canonica della Fase 4A.
- Verificare i nuovi flussi con Number/String distinti, Boolean/Null in membership, interi a 64 bit, precisione 76/scala 37, cambi schema, cancellazione e risposte fuori ordine.

L'editor condiviso continua a indirizzare le righe tramite path posizionali; il refactoring non introduce da solo ID semantici stabili. Per la Fase 4B restano necessari indirizzamento e presentazione separati dalla canonicalizzazione, integrandoli nelle sessioni del redesign invece di ricreare un'altra esperienza di editing.

## 14. Eventi e riconciliazione delle identità

Gli eventi conservano un'associazione al profilo, `connectionId`, `messageId`, tempi dell'azione/ricezione/invio e proprietà business JSON. Lo schema espone anche `originalTimestamp`, ma il mapping verso il warehouse lo esclude. Il decoder può normalizzare/correggere il timestamp e, nel percorso previsto, troncarlo a millisecondi.

`messageId` resta la chiave di merge dell'evento; il decoder ne genera uno se assente. Non è automaticamente l'identità dell'azione business e non è un Order ID o Product ID. `connectionId` fornisce provenienza, ma non compone la chiave primaria `message_id`. KPID identifica il profilo; la sua presenza non certifica una deduplicazione della stessa azione registrata da browser, server e import.

I dati sono quindi una base per raw behavior e filtri per record. Per same-object, conteggi alla grana corretta, primo/ultimo, percorsi e assenze servono mappings e contratti ulteriori: quale azione conta, come si deduplica, quale ordine è affidabile a timestamp uguale, quale storia è osservabile. `profile-roles` non colma queste differenze e il redesign non aggiunge da solo tali certificazioni.

Riferimenti: [schema Event](../core/internal/schemas/event.go#L12), [mapping](../core/internal/datastore/events.go#L20), [chiave di merge](../core/internal/datastore/events.go#L138), [generazione messageId](../core/internal/collector/decoder.go#L705), [tempo](../core/internal/collector/decoder.go#L749), [tabella PostgreSQL](../warehouses/postgresql/tables/events.sql#L89), [tabella Snowflake](../warehouses/snowflake/tables/events.sql#L76).

## 15. Confronto con la specifica congelata e il Canonical AST

| Capacità o vincolo | Stato attuale | Conseguenza per la Fase 4B |
|---|---|---|
| Proprietà scalari e operandi tipizzati | Implementati; semantiche di valore disponibili. | Base riusabile per V, con risoluzione del dominio e dei reference. |
| Ruoli delle proprietà correnti | Previsti con `profile-roles`. | Integrare le assegnazioni senza inventare un secondo mapping scollegato. |
| Confronto fra due proprietà | Il filtro pubblico corrente contiene operandi letterali. | Il relativo confronto V è una capacità da aggiungere. |
| Aggregazioni comportamentali | Assenti dal linguaggio pubblico. | Count/sum/average/distinct devono avere scope, grana e requisiti di evidenza. |
| E, K, P, T | Non rappresentati da `FilterRule`. | Nuove forme chiuse; non semplice aggiunta di operatori alla foglia corrente. |
| Corpi locali e ownership | Assenti nel filtro per record. | Conservare selezione prima della verifica e legami a Self/key senza export verso condizioni sorelle. |
| T, calendari, finestre e deadline | Non sono concetti del filtro attuale. | Definirne l'esecuzione rispettando le semantiche già congelate. |
| Contratti autorevoli e Certified Derived Evidence | Assenti come sottosistemi del filtro. | Risolvere contratti e consumer; disponibilità distinta dalla validità e dall'esito. |
| AND/OR | Presenti, con risultato booleano Go. | Implementare le tabelle ternarie; non dedurle dalle normali condizioni SQL. |
| Assenza, null, vuoto | Distinguibili in JSON; alcuni test legacy li aggregano. | Esplicitare i test target senza copiare etichette o fallback legacy. |
| Filtro senza condizioni | `nil`. | Mapping concettuale ad `AllProfiles`, distinto da empty group invalido. |
| Gruppi singleton | Ammessi e conservati. | Eliminabili nella forma canonica, preservando le informazioni utili all'editor. |
| Profondità root | Quattro gruppi. | Target tre; verificare rappresentabilità e filtri persistiti. |
| Limite delle regole | Cento discendenti, gruppi inclusi. | Non attribuire al target questo conteggio senza una decisione sui suoi limiti di implementazione. |
| Equality | Albero ordinato; equivalenza dei valori JSON per kind. | Non equivale alla canonical equality della Fase 4A. |
| Preview e sessione di authoring | Previsti dal redesign. | Riutilizzare il flusso; aggiungere evidenze e diagnostica richieste dal target. |

### La differenza più rilevante resta UNKNOWN

La Frozen §19 distingue assenza certa, null esplicito, testo vuoto e campo non osservabile. Un normale confronto `= valore` o `≠ valore` su missing/null produce `UNKNOWN`. Solo `MATCH` include il profilo. Un'esecuzione fallita non è un quarto truth value e non deve essere convertita in `UNKNOWN` o falso.

Il legacy JSON restituisce invece true per `is not [100]` su un path assente, e false per `is [100]`. Il solo fatto di avere eliminato SQL NULL dai predicati JSON non lo rende quindi conforme al target: la logica ternaria SQL e quella semantica della Frozen sono problemi diversi. `is empty` legacy, che include assenza, null e contenitori vuoti, non coincide con il solo test target del testo vuoto.

Una traduzione che preservi soltanto la membership per valori noti non dimostra equivalenza su dati ignoti o su costruzioni future che quantificano quelle foglie. Il documento 07 §18 e il documento 08 F6 escludono di risolvere il problema inserendo un `legacyMode`, una policy di unknown configurabile o un predicato opaco nel nuovo AST.

### Profondità e normalizzazione

`AND(a, AND(b, AND(c, AND(d,e))))` può essere appiattito senza perdere il significato. `AND(a, OR(b, AND(c, OR(d,e))))` ha invece quattro livelli alternati: non si riduce con il solo flattening. Eventuali distribuzioni booleane possono aumentare la struttura; la Fase 4A non le prescrive come policy ordinaria. L'ammissione dei filtri legacy e la loro rappresentazione canonica richiedono quindi una verifica esplicita.

I limiti target confermati sono quelli del documento 06 §7.5 e del documento 08 F4: tre livelli root, due nei corpi/where, quattro scope subordinati per owner che li ammette, posizioni 1–10, percorsi di 2–5 passi con una sola chiave in un blocco contiguo. Non sono equivalenti al semplice limite legacy di cento regole.

## 16. Confini di implementazione e superfici di authoring

`core` rimane il punto autorevole per costruire e risolvere il modello semantico. Un package interno può contenere union chiuse, costruttori controllati e riferimenti risolti; `cmd`, Admin e MCP devono entrare attraverso le API di `core`. Tipi nominali e validazione del registry devono verificare a runtime i riferimenti provenienti da schema e configurazione: i generics non dimostrano da soli autorità, compatibilità di unità o completezza dei dati.

`state.Where` e `warehouses.Expr` sono punti di riuso per la rappresentazione corrente e l'esecuzione, non la fonte normativa del nuovo dominio. Il compiler può tradurre dal modello canonico a espressioni warehouse; il modello canonico non deve dipendere dal dialect o dalla tabella fisica degli eventi. Il valutatore per record resta utile, ma il suo ritorno `bool` non basta a eseguire il nuovo modello.

La toolchain del repository è Go 1.27, con Go 1.27.2 nell'ambiente di verifica; Docker e workflow sono allineati. `core.FilterRule` mostra già una union con marker privato; il modello interno trusted richiede comunque un confine distinto dai DTO pubblici mutabili. Riferimenti: [go.mod](../go.mod#L5), [Dockerfile](../Dockerfile#L5), [FilterRule](../core/filters.go#L38).

Operatori e regole editoriali sono ancora replicati in Go e TypeScript. Il nuovo editor separato migliora il confine, ma non introduce un endpoint server di capability discovery per il modello target. Il futuro authoring AI deve produrre input da risolvere e validare in `core`; prompt, etichette e provenance non devono diventare la fonte del risultato a runtime.

### Presentazione e uguaglianza

`state.Where.Equal` confronta i figli in ordine. Il rilevamento delle modifiche dei pipeline confronta anche la serializzazione `MarshalWhere`. Non esiste una canonicalizzazione semantica generale con flattening, ordinamento e deduplicazione dei gruppi. Il redesign aggiunge fingerprint utili alla sessione, ma questi non sostituiscono il modello della Fase 4A.

La Fase 4B deve conservare presentazione e intenzione editoriale separatamente dalla canonical equality. Un hash del filtro identifica una richiesta normalizzata: da solo non rende riusabile un risultato a qualunque T, con qualunque evidenza o versione dello schema. Fonti: [Where.Equal](../core/internal/state/where.go#L34), [uguaglianza valori](../core/internal/state/where.go#L106), [change detection](../core/pipeline.go#L1634), documento 07 §§16–17 e documento 08 F7.

## 17. MCP e accesso SQL raw

Il server MCP espone informazioni sul warehouse, schema Profile/Event e query SQL read-only. La chiave deve essere del tipo MCP, appartenere a un'organizzazione abilitata ed essere limitata a un workspace. Il tool di query passa attraverso `Workspace.QueryWarehouseReadOnly` e il rate limiter; il percorso usa la configurazione MCP del warehouse.

La risposta raw è un array di righe con limite di 10 MiB; al superamento il metodo può restituire un risultato parziale insieme a un errore. PostgreSQL aggiunge una transazione read-only alla validazione SQL e richiede le assunzioni di deployment documentate; Snowflake valida il sottoinsieme SQL ammesso e richiede privilegi read-only. Queste difese non costituiscono la semantica del filtro.

La discovery può già esporre schema e tipi, comprese le semantiche presenti nel tipo. Non fornisce ancora repertori, identità business e contratti del Canonical AST. Non è stato individuato un tool MCP che costruisca o esegua quel linguaggio. Un futuro adapter semantico dovrà usare la medesima authority `core`, mantenendo distinta la capability SQL raw.

Nell'attuale percorso non è definito un protocollo di evidence explanation o un audit semantico delle decisioni. Timeout, audit esterni e configurazioni di produzione non possono essere dedotti dai soli handler MCP.

Riferimenti: [server MCP](../cmd/internal/mcp/mcp.go#L72), [tool](../cmd/internal/mcp/tools.go#L80), [risoluzione workspace](../cmd/internal/mcp/tools.go#L296), [query core e limite](../core/workspace.go#L1356), [PostgreSQL](../warehouses/postgresql/queryreadonly.go#L18), [Snowflake](../warehouses/snowflake/queryreadonly.go#L14).

## 18. Verifiche eseguite e copertura disponibile

Le verifiche usano i test esistenti, senza modifiche al codice applicativo o nuovi test. Il repository aveva file di lavoro non tracciati; non sono stati modificati. Le verifiche riguardano il commit della baseline, non un risultato di merge con i branch in revisione.

| Verifica | Esito |
|---|---|
| Selezione filtri in `core` | Superata: decoder, operatori, valori, limiti, conversioni, array e round-trip tipizzati. |
| Selezione `core/internal/state` | Superata per i test selezionati, inclusa la successiva esecuzione del round-trip JSONB su PostgreSQL. |
| `core/internal/filters` | Superata la selezione del valutatore, presenza e membership tipizzata. |
| `core/internal/datastore` | Superati i test selezionati di conversione `Where`/Expr. |
| Renderer PostgreSQL e Snowflake | Superati i test selezionati di rendering, modi strutturale/tipizzato, valori non validi e limiti decimal. |
| `TestJSONFilters/PostgreSQL` | Superato: confronto differenziale con il valutatore Go sui dati riletti dal warehouse. |
| `TestWhereTypedJSONBRoundTrip` | Superato su container PostgreSQL temporaneo. |
| `typed-filter-values.spec.ts` | **42 superati, 1 saltato**. Incluso il test Chromium su griglia, editor, salvataggio, anteprima e reload della fixture. |
| Snowflake reale | Non eseguito: configurazione `KRENALIS_SNOWFLAKE_TESTER_*` assente. Il codice e i test sono presenti; i renderer sono stati verificati localmente. |
| Admin con backend reale | Il relativo test è saltato: manca il backend predisposto da `TestAdmin`. I test HTTP locali usano risposte simulate. |
| `profile-roles` e `profiles-list-redesign` | Lettura statica delle revisioni indicate; nessuna esecuzione o attestazione del risultato integrato. |

Il primo tentativo di accesso a Docker e di avvio Chromium era bloccato dal sandbox; le verifiche dipendenti sono state ripetute con l'accesso necessario e gli esiti sopra sono quelli finali.

La copertura differenziale JSON usa fixture con attesi espliciti in `test/warehouseexpr`, condivise con i due warehouse, e una matrice più ampia in `query_json_test.go`. Il confronto effettivo verifica conteggio dei match e assenza di ID inattesi. Non è una prova esaustiva di ogni combinazione ammessa dall'API: il test differenziale costruisce `state.Where`, mentre preparazione pubblica e trasporto hanno test separati. Rimane da estendere l'analisi sistematica ai tipi ordinari, soprattutto sui valori mancanti.

Riferimenti: [fixture condivise](../test/warehouseexpr/json.go#L26), [oracle Go](../core/internal/datastore/query_json_test.go#L35), [matrice SQL](../core/internal/datastore/query_json_test.go#L358), [suite PostgreSQL](../warehouses/postgresql/jsonexpr_test.go#L139), [suite Snowflake](../warehouses/snowflake/jsonexpr_test.go#L185), [test Admin](../admin/tests/typed-filter-values.spec.ts#L540).

### Comandi di riproduzione

Dalla radice del repository, selezione Go eseguita:

```sh
go test -mod=vendor ./core ./core/internal/state ./core/internal/filters ./core/internal/datastore ./warehouses/postgresql ./warehouses/snowflake -run '^(Test_[Ff]ilter.*|Test_prepareFilterValues|Test_convert.*Filter.*|TestDecodeFilter.*|Test_resolveFilterProperty|Test_validate.*Filter.*|TestFilter.*|TestPrepare.*|TestTyped.*|TestWhere.*|Test_.*Where.*|Test_Applies.*|Test_readPropertyFrom|TestApplies.*|TestConvertWhere.*|Test_renderExpr.*|TestStructuralJSONRendering|TestJSONDecimalRendering|TestJSONExpressionErrors)$' -count=1
```

Il comando comprende il test JSONB, che richiede Docker. Verifiche PostgreSQL aggiuntive e riesecuzione del test JSONB:

```sh
go test -mod=vendor ./core/internal/state ./core/internal/datastore -run '^(TestWhereTypedJSONBRoundTrip|TestJSONFilters)$/^PostgreSQL$' -count=1
```

Da `admin/`, suite locale dell'Admin:

```sh
./node_modules/.bin/playwright test tests/typed-filter-values.spec.ts --project=pipelines.spec.ts --reporter=line
```

Nell'audit i log e la cache Go sono stati diretti a una directory temporanea. Non sono state eseguite la suite completa Admin o una qualifica di produzione.

## 19. Prestazioni e dati di produzione ancora necessari

La membership JSON numerica interpreta gli operandi costanti come `decimal.Decimal` durante le valutazioni dei record. Il valore del record viene preparato al massimo una volta nel singolo controllo, ma il parsing degli operandi può ripetersi fra record. Questo costo è visibile in [opIsIn](../core/internal/filters/applies.go#L291) e nei [benchmark](../core/internal/filters/applies_bench_test.go). Il [design Typed Filter Values §12.1](../docs/architecture/typed-filter-values-design.md#L742) registra già le misure e l'ottimizzazione rinviata; non sono state prodotte nuove misure prestazionali in questo audit.

L'eventuale preparazione di un valutatore separato è un'ottimizzazione da motivare con l'uso reale, non una decisione già presa sulla Fase 4B. Non elimina il bisogno di parsare i dati JSON letti dal record e non cambia la semantica dei filtri.

Prima di una decisione di migrazione servono dati su distribuzione dei filtri persistiti per profondità/operatori/tipi, significato degli operandi JSON precedenti, client API che inviano valori testuali, uso dei confronti negativi su assenza/null e warehouse effettivi. Per il behavioral filtering servono inoltre fonti di completezza, identità business, ordine affidabile e contratti autorevoli concreti. Questi dati non si ricavano dai soli test locali.

## 20. Vincoli operativi per la Fase 4B

Il lavoro successivo può assumere come basi già implementate la tipizzazione degli operandi, il supporto JSON attuale, la preparazione schema-aware e i codec Admin. Deve considerare come integrazioni previste i ruoli delle proprietà correnti e il flusso di authoring/anteprima del redesign, con il loro aggiornamento al formato tipizzato.

Le decisioni da rendere concrete nella specifica tecnica sono:

1. Definire il boundary fra input pubblico/editoriale, modello semantico risolto ed esecuzione, mantenendo `core` autorevole per tutte le superfici.
2. Stabilire quali filtri attuali hanno una traduzione fedele nel target, verificando operatori, assenza/null, profondità e dati persistiti; distinguere questa analisi dalla conversione di rilascio a Typed Filter Values.
3. Riusare semantiche di tipo e ruoli del profilo, aggiungendo i reference e contratti mancanti senza trasformare i path modificabili in identità semantiche immutabili implicite.
4. Implementare `MATCH`, `NO_MATCH`, `UNKNOWN` e le relative evidenze prima di attribuire conformità al target alle normali espressioni SQL.
5. Conservare grouping e intenzione editoriale dove richiesti dalle sessioni UI, separandoli dalla forma canonica e dal suo hash.
6. Verificare il risultato integrato dei tre branch con test di trasporto, editor, count/listing e SQL, oltre alle prove per i nuovi contesti comportamentali.

Non sono più giustificati blocchi basati sulla vecchia assenza di supporto JSON o dei testi normativi. Rimangono reali i vincoli di migrazione, significato, rappresentazione dei dati e disponibilità dei contratti. I requisiti della Frozen e della Fase 4A restano il riferimento per risolverli.
