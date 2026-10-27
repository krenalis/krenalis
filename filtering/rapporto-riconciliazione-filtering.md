# Krenalis — Riconciliazione del sistema di filtering

Rapporto di verifica del 25 ottobre 2026. Baseline: `typed-filter-values@e66eea846418c592b27689eff2c4ba4c06fac1b9`.

Il requisito di questa verifica è preservare il contratto del **record filtering** ed estendere l'unico `core.Filter` con lo **unified profile filtering**, nei soli contesti autorizzati. La precedenza automatica della specifica 06 sul codice esistente non è assunta. Il rapporto identifica differenze e decisioni; non approva traduzioni, formati o meccanismi di compatibilità.

## A. Baseline verificata

### A.1 Checkout e riferimenti

| Riferimento | Commit disponibile localmente | Verifica e uso in questo rapporto |
|---|---|---|
| `HEAD`, `typed-filter-values`, `origin/typed-filter-values` | `e66eea846418c592b27689eff2c4ba4c06fac1b9` | Baseline effettivamente letta ed eseguita; coincide con l'audit 10. |
| `json-filter-support`, `origin/json-filter-support` | `82e7be3a1939340cfc3374cf9f8458a63f21a43b` | Genitore diretto di HEAD, quindi incluso. |
| `profile-roles` | `f4f9101f1999c3b4c46c77d90a122af6001a6cb2` | Non antenato di HEAD; nessuna integrazione presunta. |
| `origin/profile-roles` | `0014b6843086d2ac9f3dfd574f71377627868619` | Non antenato di HEAD; lettura statica mirata del lavoro previsto. |
| `profiles-list-redesign` | `95f45859515df3b727cb712985efd13ec78e697b` | Non antenato di HEAD. |
| `origin/profiles-list-redesign` | `2f0a2fb35c9d7459f56cbaa6186431f4a8a394f1` | Non antenato di HEAD; lettura statica mirata di editor, sessioni e count. |
| `main`, `origin/main` | `3f36f0554a58da789fa80aded21204cf2bdb42da` | Riferimento di contesto; non usato come contratto della baseline. |

HEAD ha un unico genitore, `82e7be3a…`, data commit `2026-10-25T14:05:13+01:00`, subject `core, warehouses, admin: implement typed filter values`. `git status` non segnala modifiche tracciate. Esistono numerosi file non tracciati preesistenti, fra cui cache e materiale di lavoro: la pulizia del checkout si riferisce ai file tracciati, non all'assenza di questi file.

Verifiche effettuate: `git status --short --branch --untracked-files=no`, `git rev-parse HEAD`, `git show -s`, `git for-each-ref` e `git merge-base --is-ancestor <ref> HEAD`. Nessun fetch, checkout, merge o migrazione. I riferimenti `origin/*` sono fotografie locali e non attestano lo stato attuale del server remoto. La mancata discendenza attesta che quei tip non sono inclusi; la presenza delle strutture specifiche è stata controllata anche nei sorgenti.

**Non c'è codice successivo alla baseline da separare nel checkout.** Tutte le conclusioni operative senza altra indicazione riguardano `e66eea846…`. Le osservazioni sui due branch sono sempre condizionali e associate ai rispettivi commit.

### A.2 Fonti e criterio delle evidenze

Fonti progettuali lette: [03 — benchmark](/Users/marco/src/krenalis/filtering/03-benchmark-intenti-segmentazione-v1.0.md), [06 — specifica congelata](/Users/marco/src/krenalis/filtering/06-specifica-semantica-normativa-congelata-v1.0.md), [07 — progetto tecnico](/Users/marco/src/krenalis/filtering/07-ast-canonico-progetto-tecnico-completo.md), [08 — estratto finale](/Users/marco/src/krenalis/filtering/08-ast-canonico-modello-finale-estratto.md), [10 — audit](/Users/marco/src/krenalis/filtering/10-audit-filtri-attuali-typed-filter-values.md). Il documento 08 ripropone il modello finale di 07; non è un contratto concorrente. Il benchmark contiene 100 intenti, oltre a casi difficili, contrasti e segmenti completi: la presente verifica non certifica una nuova percentuale di copertura del benchmark.

Sono stati letti direttamente modello pubblico, preparazione, parser degli operandi, stato/persistenza, evaluator, compiler, entrambi i renderer, test pertinenti, chiamanti core/cmd e codec/editor Admin. Le fonti progettuali originarie 03/06/07/08 sono in UTF-16 BE, con dati accessori finali: per la lettura sono state create copie decodificate esclusivamente nella directory temporanea del rapporto. I riferimenti progettuali usano le sezioni, senza alterare i documenti originali.

Nel seguito:

- **codice**: comportamento ricostruito dai rami effettivi del sorgente; può essere una deduzione senza esecuzione del warehouse;
- **test locale**: test esistente rieseguito in questa attività;
- **storico 10**: risultato riportato dall'audit precedente, non rieseguito qui;
- **da verificare**: manca una prova sufficiente, in particolare sul servizio Snowflake reale o sui dati installati.

L'audit 10 è un punto di partenza, non sostituisce questa ricostruzione. Le sue indicazioni architetturali, compresa la frase sul modello semantico distinto, non sono trattate come decisioni aggiuntive approvate: è vincolante il requisito dell'utente di un unico `core.Filter`; la sua realizzazione interna resta aperta.

### A.3 Test eseguiti in questa attività

Ambiente: macOS 27.0.1, arm64, `go version go1.27.2 darwin/arm64`; dipendenze vendorizzate. `GOCACHE` esterna al repository. Comando, dalla root del monorepo:

```bash
GOCACHE=/private/tmp/krenalis-audit-go-cache go test -mod=vendor \
  ./core ./core/internal/state ./core/internal/filters ./core/internal/datastore \
  ./warehouses/postgresql ./warehouses/snowflake \
  -run '^(Test_[Ff]ilter.*|Test_prepareFilterValues|Test_convert.*Filter.*|TestDecodeFilter.*|Test_resolveFilterProperty|Test_validate.*Filter.*|TestFilter.*|TestPrepare.*|TestTyped.*|TestWhere.*|Test_.*Where.*|Test_Applies.*|Test_readPropertyFrom|TestApplies.*|TestConvertWhere.*|Test_renderExpr.*|TestStructuralJSONRendering|TestJSONDecimalRendering|TestJSONExpressionErrors)$' \
  -skip '^TestWhereTypedJSONBRoundTrip$' -count=1 -json \
  > /private/tmp/krenalis-filtering-reconciliation-20261025/go-unit.jsonl
```

| Package | Test principali passati | Subtest passati | Esito |
|---|---:|---:|---|
| `core` | 35 | 653 | Pass |
| `core/internal/state` | 13 | 235 | Pass |
| `core/internal/filters` | 6 | 663 | Pass |
| `core/internal/datastore` | 14 | 408 | Pass |
| `warehouses/postgresql` | 8 | 165 | Pass |
| `warehouses/snowflake` | 7 | 172 | Pass |

Totale: **83 test principali e 2.296 subtest passati**, zero fallimenti; non sono la suite completa del prodotto. Il filtro `-skip` esclude intenzionalmente il round-trip JSONB con database; le suite di esecuzione `TestJSONFilters` e `TestJSONExpressions` non sono selezionate. Nessun container, database, browser o servizio condiviso è stato avviato da questa verifica. La configurazione `KRENALIS_SNOWFLAKE_TESTER_*` non è presente nell'ambiente.

L'audit 10 §18 riporta storicamente anche PostgreSQL differenziale, round-trip JSONB e 42 test Admin passati/1 saltato. **Questi risultati non sono nuove esecuzioni**. I test correnti di rendering verificano SQL prodotto, non l'esecuzione di tutte le espressioni sui due warehouse. [Log di questa esecuzione](/private/tmp/krenalis-filtering-reconciliation-20261025/go-unit.jsonl).

## B. Contratto effettivo del record filtering

### B.1 Rappresentazioni e costruzione

| Livello | Rappresentazione effettiva | Vincoli e distinzione |
|---|---|---|
| API Go/HTTP | `core.Filter {Operator FilterLogical, Rules []FilterRule}`; foglia `FilterCondition {Property string, Operator FilterOperator, Values []json.Value}` | Root gruppo AND/OR, foglie o gruppi; nessun NOT generico. `FilterRule` ha un marker privato: all'esterno di `core` non si aggiungono implementazioni arbitrarie. |
| Costruzione Go | Composite literals, `MarshalJSON`, `UnmarshalJSON` | Non ci sono costruttori pubblici `NewFilter`/`NewFilterCondition` che attestino la validità. Serializzare o decodificare non equivale a validare contro schema. |
| Esecuzione/stato | `state.Where`, `WhereRule`, `WhereCondition {Property []string, Operator enum, Values []any}` | Operandi già convertiti a string/int/uint/float64/Decimal/time.Time/json.Value. Nessun AST comportamentale. |
| Persistenza/notifiche | JSON del `state.Where`: logici `And`/`Or`, operatori `Is`, `IsNot`, ecc.; path a segmenti; valori JSON tipizzati | Diverso dal DTO HTTP, che usa `and`/`or`, nomi con spazi e path puntato. Conversione dipendente dallo schema. |
| Warehouse | `warehouses.MultiExpr`, `BaseExpr {Column, Keys, Operator, Values, JSONComparison}` | Costruttori `NewMultiExpr` e `NewBaseExpr`; `Keys` può rappresentare chiavi letterali più ampie della sintassi pubblica. Il default di confronto JSON è strutturale; i filtri selezionano esplicitamente quello tipizzato. |
| Admin | `EditorFilter` → `Filter` → `TransportFilter` | Draft testuale con condizioni incomplete e `valueKinds`; modello tipizzato con `FilterNumber {kind:'number',text}`; trasporto mediante BigNumber e JSONbig. Non sono tre API semantiche di filtering. |

Evidenze: [modello pubblico](/Users/marco/src/krenalis/core/filters.go:23), [condizione e codec](/Users/marco/src/krenalis/core/filters.go:97), [modello stato](/Users/marco/src/krenalis/core/internal/state/where.go:28), [BaseExpr e modalità JSON](/Users/marco/src/krenalis/warehouses/warehouses.go:824), [tipi Admin](/Users/marco/src/krenalis/admin/src/lib/api/types/pipeline.ts:93). Test: `Test_Filter_JSON`, `Test_Filter_MarshalJSONDoesNotValidateSemantics`, `Test_Filter_UnmarshalJSONDoesNotValidateSemantics`, `TestConvertWhereSimple`, `TestStructuralJSONRendering`.

Esempio HTTP valido, con Number e String distinti:

```json
{"operator":"and","rules":[
  {"property":"account","operator":"is","values":[9223372036854775807]},
  {"property":"properties.status","operator":"is one of","values":[100,"100",true,null]}
]}
```

Nelle espressioni warehouse interne `Values` può anche contenere una `warehouses.Column`, che i serializer rendono come identificatore. Questa possibilità interna non è esposta da `FilterCondition.Values` e non dimostra un confronto pubblico fra proprietà già disponibile. [Serializer PG](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:562), [SF](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:600).

Schema necessario: `account` intero signed, `properties` JSON. Il corrispondente stato persistito usa `"And"`, `"Is"`, `"IsOneOf"` e `"property":["properties","status"]`.

### B.2 Algebra, assenza di filtro e limiti

| Caso | Contratto pubblico/preparazione | Dettaglio esecutivo o di trasporto |
|---|---|---|
| Filtro assente / `nil` | Nessun vincolo | `prepareFilter(nil)` → nil; `Applies(nil,...)` → true; query senza Where. Non scavalca consensi, matching o altri vincoli dell'operazione. |
| `filter=null` in query string | Rifiutato da Profiles, Events e ApplicationUsers | Parametro assente o stringa vuota non costruisce un filtro. Un puntatore nil in body pipeline rappresenta invece nessun filtro. |
| `{}` / root foglia | Non sono un filtro pubblico valido | Root deve contenere `operator` e `rules`; il filtro non è direttamente una condizione. |
| Gruppo esplicito con `rules:[]` | Può essere decodificato strutturalmente, ma `prepareFilterGroup` lo rifiuta | Non è il modo pubblico per esprimere tutti/nessuno. Internamente `Applies(And([]))=true`, `Applies(Or([]))=false`, mentre `convertWhere` rifiuta gruppi vuoti: non promuovere questi dettagli a contratto API. |
| Singleton | Valido con AND o OR, stesso significato della foglia | Non eliminato automaticamente; struttura e operatore rimangono osservabili nei round-trip e nell'uguaglianza. |
| Gruppi consecutivi con stesso operatore | Ammessi | Nessun flattening automatico. |
| Profondità | Massimo **4 gruppi**, root inclusa | Non conta le condizioni come quinto gruppo. Il decoder e la preparazione dei literal Go applicano il limite. |
| Regole | Massimo **100 nodi discendenti**, gruppi più condizioni, root esclusa | Non è un limite di 100 sole foglie né di 100 per gruppo. |
| Operandi | 0, 1, 2 o lista non vuota secondo operatore | Non emerge un massimo specifico per la lunghezza delle liste di membership; non va confuso con 100 regole. Non è attestato qui un limite globale HTTP comune a tutti gli ingressi. |
| Operatori unari | `Values == nil` | Omissione e `values:null` producono nil; `values:[]` è non-nil e viene rifiutato. Un elemento `[null]` è invece un operando Null, permesso solo dalla membership JSON. |

Il decoder ricorsivo rifiuta campi sconosciuti o duplicati, regole nil, mix `property`/`rules` e `values` sui gruppi. Non attesta operatore valido, schema o compatibilità degli operandi: questi controlli appartengono alla preparazione. Anche il marshal non applica tutti i limiti semantici. Il metodo standalone `FilterCondition.UnmarshalJSON` usa una struttura alias; non sostituisce il decoder ricorsivo dell'intero filtro.

Evidenze: [decoder](/Users/marco/src/krenalis/core/filters.go:371), [preparazione gruppi](/Users/marco/src/krenalis/core/filters.go:722), [cardinalità](/Users/marco/src/krenalis/core/filters.go:659), [Applies](/Users/marco/src/krenalis/core/internal/filters/applies.go:18), [compiler](/Users/marco/src/krenalis/core/internal/datastore/query.go:58), [Profiles HTTP](/Users/marco/src/krenalis/cmd/workspaces.go:892). Test: `Test_Filter_UnmarshalJSONLimits`, `Test_validateFilterLimits`, `Test_validateFilterRejectsNilRules`, `Test_Filter_UnmarshalJSONRejectsInvalidStructure`, `TestConvertWhereRejectsMissingRules`.

### B.3 Proprietà, path e contenitori

Il path pubblico segue `[A-Za-z_][A-Za-z0-9_]*`, con segmenti separati da punti. Non è JSONPath: niente indici, wildcard, chiavi quotate, traversal implicito degli elementi di array o map. Un oggetto di schema può essere attraversato; incontrata una proprietà JSON, la preparazione conserva quel tipo e separa il path interno dal path base necessario a leggere il record. Per esempio `properties.cart.price` richiede `properties` come proprietà base e `cart`, `price` come chiavi JSON.

La distinzione è sostanziale:

- **Object di schema**: figli dichiarati e typed; nel warehouse è appiattito in colonne. `exists` sul padre diventa OR delle presenze dei discendenti, `does not exist` AND delle assenze. Non si conserva la presenza autonoma di `{}`.
- **JSON Object/Array**: un valore JSON conserva il proprio kind. Gli oggetti sono attraversati per chiavi; `contains` su un array JSON cerca elementi immediati String o Number; non naviga oggetti contenuti.
- **Array typed**: `contains` ammesso per elementi scalari accettati da `is`, compreso JSON; esclusi Boolean e contenitori. Non esistono quantificatori o confronti fra array.
- **Map typed**: soltanto i test unari applicabili; non si accede a una chiave della map mediante il path pubblico. Map non equivale a un insieme dichiarato della specifica 07.

La sintassi di `BaseExpr.Keys` è più ampia: i test interni con chiavi contenenti punti o apici non dimostrano che tali chiavi siano indirizzabili dal JSON pubblico.

Evidenze: [path](/Users/marco/src/krenalis/tools/types/properties.go:270), [ByPath](/Users/marco/src/krenalis/tools/types/properties.go:43), [resolveFilterProperty](/Users/marco/src/krenalis/core/filters.go:347), [lettura Go](/Users/marco/src/krenalis/core/internal/filters/applies.go:537), [compilazione object/JSON](/Users/marco/src/krenalis/core/internal/datastore/query.go:100), [colonne profilo](/Users/marco/src/krenalis/core/internal/datastore/utils.go:165). Test: `Test_resolveFilterProperty`, `Test_AppliesWithNestedPath`, `Test_readPropertyFrom`, `TestConvertWhereExistsOperators`, `TestConvertWhereRejectsUnsupportedObjectOperator`.

### B.4 Preparazione, tipi e precisione

`prepareFilter` visita e valida l'albero, risolve lo schema, verifica operatori/cardinalità/valori e produce `state.Where` con path base distinti e ordinati. Le condizioni HTTP non contengono un tipo dichiarabile dall'utente separatamente dal path. Il tipo è determinato dallo schema; un path interno JSON ha invece un kind dinamico del dato, e gli operandi conservano il proprio kind.

| Proprietà o elemento | Payload HTTP ammesso | Valore preparato e limiti effettivi |
|---|---|---|
| String | JSON String | String non vuota, UTF-8 valida, senza NUL, massimo 80 rune, salvo valori ammessi dell'enum; JSON String vuota ha l'eccezione sotto. |
| UUID / IP | JSON String | Normalizzazione UUID/IP dedicata, poi uguaglianza della rappresentazione canonica. |
| Int signed/unsigned | JSON Number in sintassi intera | `ParseInt`/`ParseUint` base 10 a 64 bit, quindi `int`/`uint` su questa piattaforma a 64 bit. `"100"`, `100.0`, `1e2` non sono soglie intere ammesse. Range indipendente dalla bit size o dal range più stretto della proprietà. |
| Year | JSON Number intero | 1–9999; operatori temporali, non i quattro operatori pubblici `less/greater`. |
| Float(32/64) | JSON Number | `strconv.ParseFloat` con bit size della proprietà, risultato `float64`; arrotondamento binario previsto. I limiti Decimal 76/37 non si applicano. Non si accettano operandi `NaN`/Infinity come stringhe o JSON non valido. |
| Decimal | JSON Number | `decimal.Decimal` esatto; precisione ≤76, scala ≤37, indipendenti da precisione/scala/range più stretti della colonna. Nessun passaggio per float64. |
| DateTime | JSON String ISO8601 | `time.Time`, anno dell'istante UTC 1–9999; offset preservato nella preparazione. Il parser vendorizzato accetta anche input senza timezone, usando UTC. |
| Date | String `YYYY-MM-DD` | `time.Time`, anno 1–9999; nessuna promozione implicita a DateTime nel filtro. |
| Time | String con ora/minuto/secondo, frazione eventuale | `ParseTime`, ancorata a 1970-01-01 UTC; fino a 9 cifre frazionarie. Il parser scarta caratteri finali, non è un parser di timezone per Time. |
| JSON / suo path | String o Number; Boolean e Null soltanto nella membership | `json.Value`; String resta String, Number normalizzato esattamente entro 76/37. `100`, `100.0`, `1e2` sono numericamente equivalenti; `"100"` è distinto. String vuota ammessa; Object e Array non sono operandi pubblici. |
| Boolean | Nessun operando nei test dedicati | `is true`/`is false`, più gli unari permessi dai flag. `is [true]` non è una forma pubblica Boolean valida. |

La preparazione **non è la validazione di un nuovo valore da inserire nella proprietà**: soglie numeriche possono superare il range dei dati. Non applica automaticamente pattern, maxLength, semantica country/phone o conversioni di unità ai letterali String. L'enum ordinario verifica l'appartenenza del valore all'elenco; nel caso `array(enum)` il ramo non ripete il controllo dell'elenco dell'elemento. Queste regole non vanno sostituite incidentalmente con il decoder generale dei record, che ha un contratto diverso (per esempio rappresenta gli int64 dei record come stringhe JSON).

Sono già implementate semantiche `country`, `email`, `phone`, `url`, `money`, `percentage`, `measurement`, `duration`. Money richiede Decimal e può dichiarare una valuta; measurement/duration richiedono int, Decimal o Float Real e unità. Il filtro confronta il numero nella rappresentazione del campo: non riceve un payload `{amount,currency}`, non effettua cambio valuta, non converte secondi/minuti né produce formule. La presenza di valuta/unità nello schema è riutilizzabile ma non certifica dominio economico, additività, grana o identità business.

Lo schema **Profiles** applica restrizioni ulteriori rispetto al catalogo generico: String e relativi elementi Array/Map non possono dichiarare enum o pattern; Array/Map non possono avere elementi Array/Object/Map; gli Array non hanno vincoli di unicità o cardinalità personalizzati. Money/percentage/measurement del profilo devono essere Decimal(18,4), Duration signed Int64, Country ISO alpha-2. Questi limiti dei dati non restringono automaticamente le soglie del filtro. [Controlli dello schema Profiles](/Users/marco/src/krenalis/core/alter_profileschema.go:174), [semantiche Profiles](/Users/marco/src/krenalis/core/alter_profileschema.go:230).

Gli errori di preparazione non sono risultati di membership: Pipeline/Events/ApplicationUsers/listener restituiscono BadRequest per filtro invalido; Profiles distingue un path inesistente con Unprocessable `PropertyNotExist`, oltre agli altri errori del relativo endpoint. Il caricamento interno può fallire con un errore senza produrre un risultato di valutazione. [Profiles](/Users/marco/src/krenalis/core/workspace.go:1279), [Events](/Users/marco/src/krenalis/core/workspace.go:940).

Evidenze: [DecodeFilterOperand](/Users/marco/src/krenalis/core/internal/state/filter_operand.go:21), [controlli String](/Users/marco/src/krenalis/core/filters.go:688), [ValidateStringField](/Users/marco/src/krenalis/core/internal/util/util.go:79), [ParseTime](/Users/marco/src/krenalis/core/internal/util/util.go:15), [parser ISO vendorizzato](/Users/marco/src/krenalis/vendor/github.com/relvacode/iso8601/iso8601.go:114), [ParseString](/Users/marco/src/krenalis/vendor/github.com/relvacode/iso8601/iso8601.go:352), [semantiche](/Users/marco/src/krenalis/tools/types/semantics.go:210), [restrizioni delle semantiche](/Users/marco/src/krenalis/tools/types/Semantics.md:3), [decoder generale dei dati](/Users/marco/src/krenalis/tools/types/value.go:410).

Test locali pertinenti: `TestDecodeFilterIntegerWidths`, `TestDecodeFilterOperand`, `TestDecodeFilterDecimalOperand`, `TestDecodeFilterFloatOperand`, `TestDecodeFilterIntOperand`, `TestDecodeFilterUnsignedOperand`, `TestDecodeFilterYearOperand`, `TestPrepareTypedJSONFilter`, `Test_validateFilterValueLimits`, `Test_convertFilterWhereTemporalArrays`.

### B.5 HTTP, Admin e persistenza

Query string Profiles/Events/ApplicationUsers: parametro `filter` contenente il JSON pubblico; i body pipeline usano lo stesso modello. L'endpoint del listener riceve un filtro nel body. La preparazione server rimane necessaria anche se l'Admin ha già validato il draft.

Nell'Admin `prepareEditorFilter` elimina righe senza proprietà e gruppi diventati privi di condizioni; un draft interamente vuoto può diventare null. Una riga con proprietà scelta ma operatore/valore invalido può invece produrre errore. È una trasformazione del draft, non l'accettazione server di gruppi esplicitamente vuoti. `valueKinds` impedisce reinterpretazioni silenziose durante i cambi di proprietà o operatore. Il testo JSON `"100"` conserva String; `100` diventa Number; Boolean/Null letterali sono riconosciuti per membership.

Il codec trasporta i numeri grandi attraverso `FilterNumber.text` → BigNumber → `JSONbig.stringify`; la libreria vendorizzata riconosce BigNumber prima di `toJSON` e ne emette il testo senza virgolette. La lettura `json-bigint` usa il testo originale per numeri lunghi; la risposta è poi adattata da `decodeFilter`. La preservazione dei letterali non elimina l'arrotondamento deliberato delle proprietà Float.

Evidenze Admin: [preparazione](/Users/marco/src/krenalis/admin/src/lib/core/pipeline.ts:2044), [parser](/Users/marco/src/krenalis/admin/src/utils/filters.ts:327), [trasporto e risposta](/Users/marco/src/krenalis/admin/src/utils/filters.ts:432), [HTTP](/Users/marco/src/krenalis/admin/src/lib/api/call.ts:19), [stringify vendorizzato](/Users/marco/src/krenalis/admin/node_modules_vendor/json-bigint/lib/stringify.js:218), [parse vendorizzato](/Users/marco/src/krenalis/admin/node_modules_vendor/json-bigint/lib/parse.js:204). Suite disponibile: [typed-filter-values.spec.ts](/Users/marco/src/krenalis/admin/tests/typed-filter-values.spec.ts:96), letta ma non rieseguita qui.

Le pipeline persistono il filtro in `pipelines.filter` JSONB e lo includono nelle notifiche di stato. Scrittura, caricamento e conversione verso il DTO usano `InSchema`; i DateTime persistiti sono normalizzati in UTC, mentre la conversione immediata di un `Where` preparato in DTO preserva il suo offset. Il loader verifica struttura, path, numero degli operandi, kind e limiti numerici; **non riesegue integralmente la preparazione pubblica** (profondità 4, 100 nodi, non-vuoto, catalogo operatore/tipo, limite String/enum). Non significa che tali forme siano ammesse dalla API: rappresenta un diverso confine di fiducia interno.

Il commit introduce operandi JSON tipizzati senza lettore precedente, versione del formato o convertitore di rilascio. Una String JSON numerica non vale più simultaneamente come testo e numero. Il [RELEASE-UPGRADE.md](/Users/marco/src/krenalis/core/internal/initdb/RELEASE-UPGRADE.md:8) richiede esplicitamente una decisione sui filtri installati prima del rilascio. Questo problema di passaggio a typed values esiste già e va distinto dalla futura estensione comportamentale; non è attestabile la migrazione dei dati reali dal solo codice.

Evidenze: [serializer](/Users/marco/src/krenalis/core/internal/state/where.go:249), [valori temporali persistiti](/Users/marco/src/krenalis/core/internal/state/where.go:312), [loader](/Users/marco/src/krenalis/core/internal/state/where.go:348), [caricamento pipeline](/Users/marco/src/krenalis/core/internal/state/load.go:617), [notifiche](/Users/marco/src/krenalis/core/internal/state/keep.go:629), [scrittura](/Users/marco/src/krenalis/core/pipeline.go:837), [conversione inversa](/Users/marco/src/krenalis/core/filters.go:200). Test locali: `TestWhereTypedPersistence`, `TestWhereTypedPersistenceValidation`, `TestTypedFilterIntegerRoundTrip`, `TestTypedFilterJSONRoundTrip`, `TestFilterDateTimeOffsetRoundTrip`; JSONB reale solo storico 10.

### B.6 Uguaglianza e rilevamento delle modifiche

`Where.Equal` confronta ricorsivamente posizione, operatore e struttura. Non commuta AND/OR, non elimina duplicati o singleton e non ordina membership. `WhereCondition.Equal` confronta numericamente Decimal e Number JSON dello stesso kind, decodifica String JSON e distingue Number da String; per altri valori usa `reflect.DeepEqual`, anche su `time.Time`. Due istanti uguali con rappresentazioni/location diverse non hanno quindi necessariamente questa uguaglianza strutturale.

Esistono due consumatori distinti: il collector confronta `Where.Equal` per aggiornare la pipeline di destinazione; `Pipeline.changed` confronta i byte di `MarshalWhere` con il nuovo filtro persistito. Non c'è un'unica canonical equality pubblica o un hash semantico. Canonicalizzare non può essere introdotto come una modifica invisibile a questi meccanismi senza valutarne gli effetti operativi.

Evidenze: [Where.Equal](/Users/marco/src/krenalis/core/internal/state/where.go:34), [WhereCondition.Equal](/Users/marco/src/krenalis/core/internal/state/where.go:106), [collector](/Users/marco/src/krenalis/core/internal/collector/destinations.go:379), [Pipeline.changed](/Users/marco/src/krenalis/core/pipeline.go:1635). Test: `Test_Where_Equal`, `Test_WhereCondition_Equal`, `TestWhereJSONSemanticEquality`.

### B.7 Contesti e chiamanti principali

Tutti i percorsi attuali sotto elencati hanno soltanto record filtering. La colonna finale descrive **il perimetro richiesto**, non un supporto già implementato.

| Contesto / ingresso | Record e validazione | Esecuzione effettiva | Implicazione dell'estensione richiesta |
|---|---|---|---|
| `GET /profiles` → `Workspace.Profiles` | Profilo unificato; `prepareFilter` contro `ws.ProfileSchema` | `Store.Profiles` → Where → Expr → SQL; proprietà restituite separate da quelle del filtro; first/limit della risposta | Repertorio completo autorizzato. |
| Pagina Profiles Admin | Profili; nella baseline `useProfiles` passa **null** | API Profiles, senza authoring del filtro in questa pagina | Repertorio completo autorizzato; authoring previsto nel redesign, non presente qui. |
| Destination / Application / User | Profilo unificato; filtro validato nel pipeline contro input schema, poi allineamento con workspace in `ProfileRecords` | `exportProfiles` → `Query{Where:pipeline.Filter}` → SQL, prima di trasformare; matching/esportazione App e consensi aggiungono altri vincoli | Repertorio completo autorizzato sulla popolazione Profiles, non sull'utente già trasformato per l'app. |
| Destination / Database / User | Come sopra | Selezione SQL dei profili, poi trasformazione e writer database | Repertorio completo autorizzato. |
| Destination / FileStorage / User | Come sopra; input schema intero del profilo | Selezione SQL, ordinamento eventualmente configurato, writer file; senza trasformazione | Repertorio completo autorizzato. |
| Source / Application / User | Utente importato; schema sorgente, normalizzazione/validazione input | Reader applicazione ed `importUsers`, filtro Go prima della trasformazione | Soltanto record filtering. |
| Source / FileStorage / User | Record utente del file; schema input e controlli del reader | `run_import.go` → `Applies`, prima della trasformazione | Soltanto record filtering. |
| Source / Database / User | Risultato della query configurata | `core.Filter` **non ammesso**; il database filtra tramite query; `validatePipelineToSet` rifiuta filtro non-nil | L'estensione richiesta non approva la rimozione di questo divieto esistente. |
| Source / SDK o Webhook / User | Evento da cui ricavare identità; schema evento sostituisce input schema esplicito | Collector `Applies` sull'evento, poi trasformazione in identità | Soltanto record filtering, anche se il target si chiama User. |
| Source / SDK o Webhook / Event | Evento | Collector `Applies` prima di pubblicare verso warehouse | Soltanto record filtering. |
| Destination / Application / Event | Evento collegato dalla sorgente; schema evento | Collector `Applies` prima dell'invio alla pipeline app | Soltanto record filtering: il ruolo Destination da solo non autorizza il repertorio esteso. |
| `GET /events` → `Workspace.Events` | Evento conservato; schema `schemas.Event` | SQL sul warehouse | Soltanto record filtering; selezionare eventi non seleziona profili in base alla storia. |
| `POST /events/listeners` → `CreateEventListener` | Evento in arrivo; schema `schemas.Event` | `Observer` → Go `Applies` prima della consegna al listener | Soltanto record filtering. |
| `GET /connections/{id}/users` → `ApplicationUsers` | Utente dell'applicazione; schema fornito e allineato al sorgente | Reader applicazione, normalizzazione, Go `Applies` | Soltanto record filtering; non confondere con export di profili verso app. |
| `GET /profiles/{kpid}/events` | Evento, con condizione su KPID costruita internamente | `cmd.ProfileEvents` costruisce `core.Filter` e richiama `Workspace.Events` | Soltanto record filtering; KPID nel filtro non introduce quantificazione comportamentale. |
| Anteprima trasformazioni nell'Admin | Record input della pipeline; filtro preparato dall'editor | ApplicationUsers oppure Profiles in base alla sorgente dell'anteprima | Dovrà rispettare il medesimo contesto del pipeline, non la mera disponibilità dell'editor condiviso. |

Riferimenti verificabili: [rotte](/Users/marco/src/krenalis/cmd/endpoints.go:36), [Profiles core](/Users/marco/src/krenalis/core/workspace.go:1251), [Profiles Admin](/Users/marco/src/krenalis/admin/src/components/routes/Profiles/useProfiles.ts:124), [Events core](/Users/marco/src/krenalis/core/workspace.go:914), [listener](/Users/marco/src/krenalis/core/workspace.go:766), [ProfileEvents](/Users/marco/src/krenalis/cmd/workspaces.go:850), [ApplicationUsers](/Users/marco/src/krenalis/core/connection.go:225), [reader](/Users/marco/src/krenalis/core/internal/connections/application.go:709), [validazione pipeline](/Users/marco/src/krenalis/core/pipeline_validate.go:154), [import](/Users/marco/src/krenalis/core/run_import.go:178), [collector](/Users/marco/src/krenalis/core/internal/collector/collector.go:844), [observer](/Users/marco/src/krenalis/core/internal/collector/observer.go:177), [export](/Users/marco/src/krenalis/core/run_export.go:64), [ProfileRecords](/Users/marco/src/krenalis/core/internal/datastore/store.go:458), [anteprime](/Users/marco/src/krenalis/admin/src/components/routes/PipelineWrapper/PipelineTransformation.tsx:1540).

Il filtro delle esportazioni è applicato durante la lettura, perciò queste pipeline non hanno uno step metrico Filter separato. Questo non significa che non filtrino. [Pipelines.md, repertorio](/Users/marco/src/krenalis/core/Pipelines.md:13) e [step](/Users/marco/src/krenalis/core/Pipelines.md:99) descrivono la distinzione, coerente con il codice.

Oggi `prepareFilter` riceve soltanto filtro/schema, senza un livello di capability operativo. Il controllo di ammissibilità della pipeline è separato e in particolare vieta Source/Database/User. Non esiste ancora una validazione Record/Unified da riusare già completa: introdurla è un requisito dell'estensione, non un comportamento presente.

### B.8 Branch previsti, tenuti separati dalla baseline

**`origin/profile-roles@0014b684…`**: `core/workspace.go:55` definisce `ProfileRoleAssignments` con firstName/lastName/email/country/photo. `core/alter_profileschema.go:292` verifica path, esistenza, unicità della proprietà assegnata e compatibilità di tipo/semantica. È pertinente ai ruoli del profilo corrente, ma non fornisce ruoli di evento, business identity, grana, completezza, contratti autorevoli o revisioni immutabili del significato. L'assegnazione modificabile di un path non è automaticamente una `SemanticRef` versionata di 07 §3. La lettura mirata di questi sorgenti è stata ripetuta qui; i test del branch non sono stati eseguiti.

**`origin/profiles-list-redesign@2f0a2fb…`**: aggiunge editor condiviso, draft/applicazione/anteprima, count e cronologia di sessione. Fonti lette: `admin/src/components/routes/Profiles/useProfiles.ts:428`, `profilesFilterSession.ts:5`, `core/workspace.go:1228` (`ProfileCount`). La cronologia ha 50 voci; il fingerprint normalizza l'operatore dei singleton per il risultato ma non implementa la canonicalizzazione di 07. `profilesSequence` coordina pagine/richieste, non è una primitive P. Count e listing non costituiscono di per sé una valutazione unica a T o una diagnostica UNKNOWN.

Il redesign usa ancora `validateFilter`/conversione e payload della base precedente; l'utente ne ha già previsto il porting al typed filtering. È un'integrazione da verificare, non una regressione già entrata nel checkout. Nessuna conclusione sul risultato finale del merge o dei test integrati viene anticipata.

Riproduzione delle fonti di branch: `git show 0014b6843086d2ac9f3dfd574f71377627868619:core/workspace.go` e `git show 2f0a2fb35c9d7459f56cbaa6186431f4a8a394f1:core/workspace.go`, analogamente per i file indicati. Le linee di branch non puntano ai file diversi presenti nel checkout.

## C. Matrice dettagliata degli operatori e dei tipi

### C.1 Applicabilità pubblica e operandi: tutti i 26 operatori

Legenda tipi: **N** = Int signed/unsigned, Float, Decimal; **S** = String, compresi i tipi con elenco di valori; **S0** = String senza elenco; **D** = DateTime, Date, Time; **Y** = Year; **U** = UUID, IP; **J** = JSON o path interno JSON; **B** = Boolean; **A(T)** = Array di T; **M** = Map; **O** = Object di schema. I flag della proprietà si applicano alla proprietà risolta, non a una nozione generale di "campo potenzialmente assente".

I numeri delle righe identificano gli operatori anche nelle matrici successive. L'elenco è ricavato dalle costanti pubbliche, non da un catalogo ipotizzato del nuovo sistema.

| # | Nome pubblico | Proprietà ammesse dal server | Numero e tipo degli operandi | Positivo/negativo e significato su valori noti |
|---|---|---|---|---|
| 01 | `is` | N, S, D, Y, U, J | 1 letterale del tipo; J: String o Number | Uguaglianza; JSON stesso kind. |
| 02 | `is not` | Come 01 | Come 01 | Complemento Go di 01; SQL ordinario usa `<>`. |
| 03 | `is less than` | N, S0, J | 1; J String o Number | `<`. |
| 04 | `is less than or equal to` | N, S0, J | Come 03 | `<=`. |
| 05 | `is greater than` | N, S0, J | Come 03 | `>`. |
| 06 | `is greater than or equal to` | N, S0, J | Come 03 | `>=`. |
| 07 | `is between` | N, S0, D, Y, J | 2; J due String o due Number | Intervallo inclusivo; gli estremi non vengono ordinati o scambiati. Eccezione temporale C.4.5. |
| 08 | `is not between` | Come 07 | Come 07 | Complemento Go/JSON di 07; SQL ordinario `NOT BETWEEN`. |
| 09 | `contains` | S, J, A(N/S/D/Y/U/J) | 1 del tipo dell'elemento; J String o Number | Sottostringa per testo; appartenenza esistenziale a elementi immediati per array. |
| 10 | `does not contain` | Come 09 | Come 09 | Complemento Go/JSON di 09; SQL ordinario predicato negativo. |
| 11 | `is one of` | N, S, D, Y, J | ≥1; stesso tipo ordinario; J lista mista String/Number/Boolean/Null | Appartenenza del valore scalare, non contenimento di un array. |
| 12 | `is not one of` | Come 11 | Come 11 | Complemento Go/JSON di 11; SQL ordinario `NOT IN`. |
| 13 | `starts with` | S0, J | 1 String | Prefisso letterale. |
| 14 | `ends with` | S0, J | 1 String | Suffisso letterale. |
| 15 | `is before` | D, Y | 1 String temporale o Number Year | `<` temporale/anno; **rifiutato su JSON**. |
| 16 | `is on or before` | D, Y | Come 15 | `<=` temporale/anno. |
| 17 | `is after` | D, Y | Come 15 | `>` temporale/anno. |
| 18 | `is on or after` | D, Y | Come 15 | `>=` temporale/anno. |
| 19 | `is true` | B, J | 0, `Values==nil` | Boolean vero; non truthiness. |
| 20 | `is false` | B, J | 0 | Boolean falso; non complemento di 19 su missing/null. |
| 21 | `is empty` | S, J, A, M | 0 | Missing/null oppure valore di contenitore/testo vuoto, nei casi sotto. O escluso. |
| 22 | `is not empty` | Come 21 | 0 | Complemento Go/JSON di 21; SQL ordinario come sotto. |
| 23 | `is null` | Qualsiasi tipo se Nullable, oppure J | 0 | Null presente in Go/JSON; SQL ordinario `IS NULL`. O nullable ammesso dalla preparazione ma non dal compiler appiattito. |
| 24 | `is not null` | Come 23 | 0 | Complemento Go/JSON di 23, include missing; SQL ordinario `IS NOT NULL`. |
| 25 | `exists` | Qualsiasi tipo se ReadOptional, oppure un path **interno** J | 0 | Presenza del path; SQL ordinario presenza di valore non-NULL. |
| 26 | `does not exist` | Come 25 | 0 | Complemento di 25; O appiattito ha il contratto sui discendenti. |

Precisazioni di applicabilità:

- Boolean ordinario non usa `is`/`is not` né membership; UUID/IP non hanno membership o range; Year usa before/after anziché less/greater.
- `contains` su S con elenco è ammesso, ma l'operando deve essere uno dei valori ammessi interi dell'enum. Non basta che sia una sottostringa valida. `starts/ends`, ordine e intervalli non sono ammessi su S con elenco.
- `is empty`/`is not empty` su S con elenco sono rifiutati se l'elenco non vuoto esclude `""`. String ordinaria senza elenco non accetta `""` come letterale di uguaglianza, anche se può contenere un dato vuoto e dispone del test `is empty`.
- La regola `null se Nullable` non comporta che ogni schema di ogni contesto ammetta Nullable: le proprietà dello schema Profiles devono essere ReadOptional e non Nullable; SQL NULL delle colonne Profiles significa assenza. JSON Null rimane un valore presente.

Evidenze comuni della matrice: [26 costanti](/Users/marco/src/krenalis/core/filters.go:144), [catalogo e switch effettivo](/Users/marco/src/krenalis/core/filters.go:533), [cardinalità e valori](/Users/marco/src/krenalis/core/filters.go:659), [decode](/Users/marco/src/krenalis/core/internal/state/filter_operand.go:25), [restrizioni schema Profiles](/Users/marco/src/krenalis/core/alter_profileschema.go:154). Test locali: `Test_validateFilter`, `Test_validateJSONFilters`, `TestPrepareTypedJSONFilter`, `Test_validateFilterValueLimits`, `TestDecodeFilterOperand`.

### C.2 Semantica JSON separata

Questa tabella riguarda **un valore JSON** e non il tipo SQL di un contenitore typed `array(json)`/`map(json)`; per questi ultimi valgono anche C.3/C.4. Missing significa path assente; Null significa JSON Null presente, non SQL NULL. **T/F** sono booleani del filtro. **E** indica String `""`, Array `[]` o Object `{}` presente. `cmp` significa confronto sul dato effettivo, non su un valore vuoto inventato. Il tipo incompatibile qui è un altro **kind JSON valido**: una configurazione con operando non ammesso è rifiutata prima dell'esecuzione.

| # / operatore | Missing | JSON Null | Valore E presente | Kind diverso/non confrontabile | Go / PostgreSQL / Snowflake |
|---|---|---|---|---|---|
| 01 `is` | F | F | `""` uguale solo a String `""`; []/{} F | F | Stesso predicato tipizzato; precisione/storage C.4.8. |
| 02 `is not` | T | T | Complemento 01 | T | Negazione del positivo reso non-null nei due SQL. |
| 03 `<` | F | F | cmp per String vuota; []/{} F | F | String con ordine UTF-8 esplicito, Number Decimal. |
| 04 `<=` | F | F | Come 03 | F | Come 03. |
| 05 `>` | F | F | Come 03 | F | Come 03. |
| 06 `>=` | F | F | Come 03 | F | Come 03. |
| 07 `is between` | F | F | String vuota confrontata coi due bordi String; []/{} F | F | `>= lower AND <= upper`, stesso kind; nessuna eccezione time.Time. |
| 08 `is not between` | T | T | Complemento 07 | T | Come 02. |
| 09 `contains` | F | F | `""` contiene String `""`; []/{} F | Number/Boolean/Object isolati F; un Array può contenere un elemento del kind richiesto | String: sottostringa; Array: uguaglianza scalare su elementi immediati; nessun unwrap ricorsivo. |
| 10 `does not contain` | T | T | Complemento 09 | Complemento 09 | Come 02. |
| 11 `is one of` | F | T solo se lista contiene Null | `""` coincide con String vuota in lista; []/{} F | F se nessun operando dello stesso kind e valore; Boolean e Null ammessi | Lista partizionata per kind; niente coercizione Number/String. |
| 12 `is not one of` | T | F se lista contiene Null, altrimenti T | Complemento 11 | Complemento 11 | Come 02. |
| 13 `starts with` | F | F | String vuota T solo con prefisso vuoto; []/{} F | F | Soltanto String; nessun cast di Number in testo. |
| 14 `ends with` | F | F | String vuota T solo con suffisso vuoto; []/{} F | F | Come 13. |
| 15 `is before` | — | — | — | — | **Rifiutato su J dalla preparazione**. |
| 16 `is on or before` | — | — | — | — | Rifiutato su J. |
| 17 `is after` | — | — | — | — | Rifiutato su J. |
| 18 `is on or after` | — | — | — | — | Rifiutato su J. |
| 19 `is true` | F | F | F | T solo per JSON Boolean true; altri kind F | SQL usa un test Boolean esplicito senza SQL NULL risultante. |
| 20 `is false` | F | F | F | T solo per JSON Boolean false; altri kind F | Non rende vero missing, Number 0 o String vuota. |
| 21 `is empty` | T | T | T | 0, false e altri scalari noti non vuoti F | Empty aggrega missing, Null, String/Array/Object vuoti. |
| 22 `is not empty` | F | F | F | 0/false noti T | Complemento 21. |
| 23 `is null` | F | T | F | F | Distingue Null presente da path assente. |
| 24 `is not null` | T | F | T | T | Include missing; non è `HasKnownValue`. |
| 25 `exists` | F | T | T | T per ogni kind presente | Nessun test di qualità del valore. |
| 26 `does not exist` | T | F | F | F per ogni kind presente | Soltanto assenza. |

Il README del datastore dichiara espressamente per **JSON** la semantica di `Applies` e dei complementi. I renderer la implementano mediante kind guard e `COALESCE(positivo,FALSE)` con negazione esterna; non usano il comportamento SQL ordinario per `<>` su NULL. Questa è un'intenzione documentata sostenuta dal codice e dalle fixture JSON. Non si estende automaticamente a tutti i tipi.

Evidenze: [dispatch Go](/Users/marco/src/krenalis/core/internal/filters/applies.go:45), [comparazione JSON](/Users/marco/src/krenalis/core/internal/filters/applies.go:109), [contains](/Users/marco/src/krenalis/core/internal/filters/applies.go:257), [membership](/Users/marco/src/krenalis/core/internal/filters/applies.go:291), [empty/null](/Users/marco/src/krenalis/core/internal/filters/applies.go:491), [PostgreSQL JSON](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:269), [Snowflake JSON](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:255), [contratto documentato](/Users/marco/src/krenalis/core/internal/datastore/README.md:64), [fixture indipendenti](/Users/marco/src/krenalis/test/warehouseexpr/json.go:26). Test locali: `TestAppliesTypedJSONFixtures`, `TestAppliesTypedMembershipDecoding`, `TestAppliesTypedMembershipPresence`, `TestTypedJSONRendering`, `TestTypedJSONOperandErrors`, `TestJSONDecimalRendering`, `TestJSONExpressionErrors`. Esecuzione differenziale sui warehouse: solo storico 10, non rifatta qui.

Una `map[string]any` con chiave presente e valore Go `nil` conserva la presenza per `Applies`, anche nei test Null/membership. Non è la stessa rappresentazione del path JSON assente. In SQL la colonna NULL rappresenta assenza: se un contesto nullable volesse conservare separatamente anche il nil del record, questa tabella non dimostra che il bit di presenza sia sopravvissuto allo storage.

### C.3 Tipi ordinari: tabella completa

**G** = risultato Go. **SQL-U** = SQL NULL/unknown, che non supera un WHERE; non è il risultato semantico UNKNOWN di 06. Nelle colonne missing/null la coppia è **G / SQL**; PG e Snowflake generano qui le forme ordinarie indicate, salvo riserve esplicite C.4. Si considera una colonna che può essere NULL, come tutte le colonne Profiles. Per empty su String/Map/Snowflake Array si assume `Column.Nullable=true`; con metadata non-nullable il renderer può omettere le guardie, vedi C.4.

`eval` indica che si applica il confronto ordinario al valore presente, con i limiti dei tipi della C.1. **Rcmp/Rord/Rcont/Rin/Rtext/Rtime** sono i comportamenti su record di tipo errato dettagliati in C.4.9; non sono un risultato trivalente.

| # / operatore | Missing G / SQL | Null esplicito G / SQL | Valore vuoto presente, quando tipo ammesso | SQL ordinario PG / Snowflake | Tipo incompatibile del record |
|---|---|---|---|---|---|
| 01 `is` | F / SQL-U | F / SQL-U | S vuota: F contro i letterali non vuoti ammessi; enum: eval | `col = a` | Rcmp |
| 02 `is not` | T / SQL-U | T / SQL-U | Complemento Go di 01; SQL sul dato noto | `col <> a` | Rcmp, poi complemento se non panic |
| 03 `<` | F / SQL-U | F / SQL-U | S0 vuota: ordine lessicografico; no empty numerico | `col < a` | Rord |
| 04 `<=` | F / SQL-U | F / SQL-U | Come 03 | `col <= a` | Rord |
| 05 `>` | F / SQL-U | F / SQL-U | Come 03 | `col > a` | Rord |
| 06 `>=` | F / SQL-U | F / SQL-U | Come 03 | `col >= a` | Rord |
| 07 `is between` | F / SQL-U | F / SQL-U | S0 vuota: eval; D non ha valore empty | `BETWEEN a AND b` | Rord; eccezione temporale C.4.5 |
| 08 `is not between` | T / SQL-U | T / SQL-U | Complemento Go di 07 | `NOT BETWEEN a AND b` | Rord |
| 09 `contains` | F / SQL-U | F / SQL-U | String/Array vuoti F, con i normali operandi non vuoti | PG `POSITION>0` o `a=ANY(col)`; SF `CONTAINS(col,a)` anche per Array: efficacia da verificare | Rcont |
| 10 `does not contain` | T / SQL-U | T / SQL-U | String/Array vuoti T | PG `POSITION=0` o `NOT(a=ANY(col))`; SF `NOT CONTAINS` | Rcont |
| 11 `is one of` | F / SQL-U | F / SQL-U | String vuota solo se enum ammette quel letterale, altrimenti F | `IN (...)` | Rin |
| 12 `is not one of` | T / SQL-U | T / SQL-U | Complemento 11 | `NOT IN (...)` | Rin |
| 13 `starts with` | F / SQL-U | F / SQL-U | String vuota F, operando vuoto non ammesso | PG `LEFT(...)=a`; SF `STARTSWITH` | Rtext |
| 14 `ends with` | F / SQL-U | F / SQL-U | Come 13 | PG `RIGHT(...)=a`; SF `ENDSWITH` | Rtext |
| 15 `is before` | F / SQL-U | F / SQL-U | Non applicabile | `col < a` | Rtime |
| 16 `is on or before` | F / SQL-U | F / SQL-U | Non applicabile | `col <= a` | Rtime |
| 17 `is after` | F / SQL-U | F / SQL-U | Non applicabile | `col > a` | Rtime |
| 18 `is on or after` | F / SQL-U | F / SQL-U | Non applicabile | `col >= a` | Rtime |
| 19 `is true` | F / SQL-U | F / SQL-U | Boolean false → F; false non è empty | `col` | Go false per tipi diversi da bool/JSON Boolean |
| 20 `is false` | F / SQL-U | F / SQL-U | Boolean false → T | `NOT col` | Come 19 |
| 21 `is empty` | T / T | T / T | String `""`, Array [], Map {} → T | PG array_length IS NULL; String/Map con guardia NULL; SF confronto col contenitore vuoto con guardia NULL | Go false per tipi non gestiti |
| 22 `is not empty` | F / F | F / F | String/Array/Map vuoti F | PG array_length IS NOT NULL; altri confronto con guardia NOT NULL; SF analogo | Complemento 21 |
| 23 `is null` | **F / T** | T / T | Valore vuoto presente F, salvo O appiattito | `IS NULL` | Go verifica nil o JSON Null, non valida schema |
| 24 `is not null` | **T / F** | F / F | Valore vuoto presente T, salvo O | `IS NOT NULL` | Complemento 23 |
| 25 `exists` | F / F | **T / F** | String/Array/Map vuoti T; O ha riserva C.4.2 | `IS NOT NULL`; O OR discendenti | Go segue soltanto il flag di presenza |
| 26 `does not exist` | T / T | **F / T** | String/Array/Map vuoti F; O ha riserva | `IS NULL`; O AND discendenti | Complemento 25 |

Le differenze di risultato intermedio SQL-U rispetto a G=false non cambiano sempre la membership: per le sole composizioni AND/OR di foglie positive, SQL-U non seleziona il record. Sono invece già osservabilmente diverse le righe negative che danno **G=true / SQL-U** e le coppie presenza/null in grassetto. Non è stata trovata una dichiarazione equivalente al README JSON che scelga una semantica ordinaria universale prevalente su entrambi gli executor.

Evidenze: [helpers Go](/Users/marco/src/krenalis/core/internal/filters/applies.go:131), [SQL ordinario PostgreSQL](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:80), [SQL ordinario Snowflake](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:91), [test Go con assenza](/Users/marco/src/krenalis/core/internal/filters/applies_test.go:43), [test SQL ordinario PG](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr_typed_test.go:57), [test SQL ordinario SF](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr_typed_test.go:57). La tabella SQL sui NULL è dedotta dalle espressioni prodotte, non da una nuova matrice differenziale eseguita su database.

### C.4 Divergenze, limiti e casi non uniformi

1. **Negativi ordinari sui NULL.** Go implementa il complemento; i renderer ordinari emettono `<>`, `NOT BETWEEN`, `NOT IN` e predicati testuali/array senza il `COALESCE` usato nel ramo JSON. Esempio: record `{}`, schema `n` Int ReadOptional, `n is not 100`: Go true, SQL su `n=NULL` SQL-U e record escluso. Sono due comportamenti esistenti; la presenza di un oracle Go per JSON non sceglie il corretto contratto ordinario. Test Go esistenti confermano esplicitamente il caso di assenza per `is not`, `not between`, `does not contain`, `not one of`.

2. **Null e presenza non conservati uniformemente.** Nel Go del record `{x:nil}`, `exists` è true e `does not exist` false; una colonna SQL NULL produce il contrario. Con record `{}`, `is null` è false in Go ma `IS NULL` true in SQL, se l'operatore è ammesso dal relativo schema nullable. Per Profiles ordinary `is null` non è normalmente ammesso, perché le proprietà non sono Nullable; `exists`/negative scalar restano invece pertinenti. Per un Object appiattito, `{o:{}}` risulta presente nel Go, ma se tutti i discendenti sono NULL il compiler lo considera assente. È un limite di rappresentazione documentato, non il significato di JSON `{}`: quest'ultimo rimane un valore presente. [Compiler](/Users/marco/src/krenalis/core/internal/datastore/query.go:124), [README](/Users/marco/src/krenalis/core/internal/datastore/README.md:80).

3. **Object nullable: ammissione e compilazione diverse.** `prepareFilterCondition` ammette `is null`/`is not null` su qualsiasi proprietà Nullable, anche Object. `convertWhere`, quando l'Object è rappresentato soltanto da colonne discendenti, ammette solo exists/not-exists e restituisce errore sugli altri operatori. Non implica un problema nell'attuale schema Profiles, che vieta Nullable; è una differenza reale fra la grammatica RF generale e il percorso SQL appiattito. `is empty` su Object, invece, viene già rifiutato dalla preparazione, come correttamente rileva l'audit 10. Test compiler: `TestConvertWhereRejectsUnsupportedObjectOperator`.

4. **DateTime e precisione temporale.** Go usa `time.Time.Equal/Before/After` sugli istanti. La preparazione conserva l'offset e la persistenza lo normalizza UTC, ma i literal SQL ordinari formattano l'ora senza offset: PG fino a microsecondi, Snowflake fino a nanosecondi. `12:00+02:00` è reso come `12:00`, non `10:00Z`; i test di regressione lo verificano espressamente. Il test tutela il rendering esistente, non prova che sia una scelta semantica approvata per il modello nuovo. La membership reale dipende anche da tipo/precisione storage e sessione; non è stata provata qui con database. [PG serializeValue](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:584), [SF](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:622), [test offset Go](/Users/marco/src/krenalis/core/filters_typed_test.go:22).

5. **Range temporali invertiti, differenza aggiuntiva rispetto all'audit 10.** Nessun controllo pubblico ordina o rifiuta i bordi invertiti. Per time.Time Go usa `v.After(lower) && v.Before(upper) || v.Equal(lower) || v.Equal(upper)`: un endpoint coincide anche quando lower > upper. SQL emette un normale BETWEEN e non include quell'endpoint. Esempio Date, `v=2026-10-20`, range `[2026-10-20,2026-10-10]`: Go true; SQL `v>=lower AND v<=upper` false. Il negativo inverte questa divergenza. Per Int/Decimal/String/JSON il Go usa invece la normale congiunzione dei bordi. È una deduzione da codice; non è coperta da uno specifico test di estremi invertiti individuato o eseguito qui. [Go](/Users/marco/src/krenalis/core/internal/filters/applies.go:230), [PG](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:111), [SF](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:122).

6. **Array ordinari in Snowflake.** Per tipi Array non-JSON il renderer emette `CONTAINS(col,operand)`/`NOT CONTAINS`, mentre PG usa `= ANY`. I test fissano anche `CONTAINS("VALUE",1e75)` per un array Decimal. Inoltre il ramo passa `c.Type` (Array), non `c.Type.Elem()`, a `serializeValue`: per un operando time.Time di array(DateTime/Date/Time), lo switch del serializer non trova un kind temporale e scrive il literal vuoto `''`. Per esempio `dates contains ["2026-10-20"]`, con Array(Date), produce `CONTAINS("DATES", '')`, mentre Go confronta la data corretta e PG serializza il tipo elemento. Questo secondo rilievo è una deduzione deterministica dal sorgente, distinta dalla verifica ancora mancante della funzione SQL sugli Array. I test citati fissano il testo per Decimal, non provano il caso temporale. Sono prove del testo prodotto, non di un'operazione esistenziale Array effettivamente eseguita. Senza servizio reale non si attesta qui né il suo successo né una membership equivalente. Il ramo `array(json)` ha invece una costruzione dedicata con FILTER sugli elementi e va tenuto separato. [Renderer SF](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:132), [test Array Decimal](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr_test.go:258), [ramo JSON](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:505).

7. **Empty dei contenitori e metadata Nullable.** PG Array usa `array_length(...,1) IS NULL` per empty anche senza la guardia del flag Nullable; SF usa confronto con `ARRAY_CONSTRUCT()` e aggiunge `IS NULL OR` solo se Nullable. String e Map nei due renderer hanno guardie analoghe condizionate dal flag. Su colonne dichiarate non-nullable ma contenenti SQL NULL non si può generalizzare la tabella C.3. Per Profiles il mapping imposta tutte le colonne Nullable=true, quindi il caso di tabella è quello pertinente. Nei contenitori typed JSON il dispatcher passa nel renderer JSON, ma gli unari di Array/Map sono gestiti come contenitori SQL, prima del predicato su un valore JSON. [PG contenitori JSON](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:285), [SF](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:271).

8. **Testo, Float e precisione/storage.** JSON String è confrontato in ordine byte UTF-8: Go `strings.Compare`, PG `COLLATE "C"`, SF `COLLATE(...,'utf8')`. Il renderer ordinario non applica quella collazione: parità di ordine/equality Unicode per tutte le configurazioni dei warehouse non dimostrata. I dati Float non-Real possono contenere NaN/Infinity, mentre le soglie pubbliche sono finite: Go su NaN rende false i confronti ordinati e true la diversità contro un numero finito; non è attestata una tabella differenziale warehouse. La decodifica dati consente questi valori esplicitamente in [types.value](/Users/marco/src/krenalis/tools/types/value.go:492). Per Decimal/Number JSON la soglia esatta 76/37 non recupera valori arrotondati prima del confronto: il codice SF tratta il valore JSON conservato tramite TO_JSON/DECFLOAT e soglie oltre 38 cifre significative con bordi corretti e equality impossibile; il limite del dato conservato è documentato nel README, ma non verificato live qui. PG confronta JSON Number come numeric. Anche NUL rimosso dall'encoder PG e presenza dell'Object appiattito sono limiti documentati di storage. [PG confronto JSON](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:249), [SF numeric](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:557), [limiti documentati](/Users/marco/src/krenalis/core/internal/datastore/README.md:82).

9. **Tipo incompatibile: configurazione e dato non sono lo stesso caso.** Un operando incompatibile viene rifiutato dalla preparazione. Un kind JSON non confrontabile è invece un dato JSON lecito e produce F/T secondo C.2. Un record ordinary che viola lo schema non ha una policy semantica unitaria nell'evaluator, che assume input preparati/normalizzati:

| Codice della C.3 | Comportamento Go su dato non conforme | Esempio / evidenza |
|---|---|---|
| Rcmp | Equality generica può dare false; i rami Decimal/time.Time/JSON fanno assertion sul tipo dell'operando e possono panic | `opIs`, linea 131. |
| Rord | Se il tipo runtime è riconosciuto, assertion su operando del medesimo tipo; può panic. Se non riconosciuto, false | `opIsLessThan` e analoghi, linea 146; `opIsBetween`, linea 230. |
| Rcont | String/JSON/Array hanno percorsi dedicati; tipo non gestito false, mismatch negli elementi/operandi può panic | `opContains`, linea 257. |
| Rin | Tipi riconosciuti fanno assertion sugli elementi della lista; altri false, nil gestito separatamente per Null JSON | `opIsIn`, linea 291. |
| Rtext | nil o tipo non testuale false; assertion per operand del ramo String può panic se il dato viola lo schema | `opStartsWith`, linea 407; `opEndsWith`, linea 426. |
| Rtime | nil false; Year usa int, gli altri casi fanno assertion time.Time | `opIsBefore` e analoghi, linea 445. |

I test Boolean, empty/null/presenza non validano il tipo declared del record: controllano le forme riconosciute o la presenza della chiave. Il flusso applicativo normale normalizza/valida gli input prima di `Applies`; un comportamento interno su record illecito non è automaticamente una promessa pubblica da perpetuare. I warehouse hanno tipi di colonna e proprie conversioni: non esiste una prova che un record ordinary malformato arrivi con lo stesso tipo a tutti gli executor. La specifica 06 §19.2 richiede di trattarlo come evidenza invalida, senza conversioni di comodo; la modalità concreta non è già presente nell'evaluator booleano.

## D. Matrice di riconciliazione

### D.1 Lettura delle classificazioni

**Equivalente** indica equivalenza entro il dominio esplicitato, non una prova su dati illeciti o su ogni executor. **Parziale** indica un sottoinsieme traducibile o una disponibilità incompleta. **Incompatibile** indica differenza effettiva di significato, forme o limiti, senza scegliere quale contratto cambiare. **Da verificare** indica che manca una definizione o una prova sufficiente. Più classificazioni possono riguardare parti diverse della stessa riga.

Le colonne delle due matrici sotto coprono: funzione, comportamento attuale, significato 06, rappresentazione 07, tipo della differenza, compatibilità, impatto sui filtri e parti documentali interessate, evidenze. I riferimenti C01–C26 rinviano ai comportamenti completi della sezione C; non li riducono al solo caso di dati noti.

### D.2 Corrispondenza di ciascuno dei 26 operatori

| Funzionalità | Record filtering effettivo | Significato 06 | Forma 07 | Tipo di differenza | Compatibilità | Impatto sui filtri esistenti | Parti potenzialmente interessate | Evidenze |
|---|---|---|---|---|---|---|---|---|
| C01 `is` | Equality tipizzata con letterale; missing false; JSON kind distinto | §8: equality su valori noti; §19.2: missing/null UNKNOWN | §4.1 `EqualLiteral(Equal)` | Sintassi; comportamento | Parziale: equivalente su scalari noti compatibili | La membership positiva isolata può coincidere anche quando il risultato diagnostico cambia; non copre JSON dinamico o ogni tipo ordinary | 06 §§8,19; 07 §§2,4,14,18 | C.1–C.4; `opIs`; test `Test_Applies`, `TestAppliesTypedJSONFixtures` |
| C02 `is not` | Complemento in Go/JSON, SQL ordinary `<>` | Diversità solo di valori noti, UNKNOWN preservato | `EqualLiteral(Different)` | Comportamento; divergenza executor preesistente | Incompatibile su missing/null per Go/JSON; equivalente sul sottoinsieme noto conforme | Profili prima inclusi dal negativo JSON sarebbero esclusi; ordinary dipende già dall'executor | 06 §§8,19,20.1; 07 §§4,14,18 | C.2/C.3, X1/X2; dispatch Go, renderer |
| C03 `is less than` | N, testo S0 e String/Number JSON | §8 limita ordine a numeri/quantità/date compatibili | `OrderedLiteral(Less)`; §4.1.1 non dà istanza a testo libero | Nomenclatura; tipo; comportamento su ignoto | Parziale; **incompatibile per ordine testuale** nel catalogo progettato | Perdita delle query lessicografiche se escluso; JSON richiede definizione del dominio | 06 §§8,19; 07 §§2,4,18 | C03; `opIsLessThan`; X7 |
| C04 `is less than or equal to` | Come C03, bordo incluso | Come C03 | `OrderedLiteral(LessOrEqual)` | Come C03 | Parziale | Come C03, preservando il bordo | Come C03 | C04; helper corrispondente |
| C05 `is greater than` | Come C03 | Come C03 | `OrderedLiteral(Greater)` | Come C03 | Parziale | Come C03 | Come C03 | C05; helper corrispondente |
| C06 `is greater than or equal to` | Come C03, bordo incluso | Come C03 | `OrderedLiteral(GreaterOrEqual)` | Come C03 | Parziale | Come C03, preservando il bordo | Come C03 | C06; helper corrispondente |
| C07 `is between` | Due estremi inclusi; testo ammesso; bordi invertiti accettati; eccezione Go time.Time | §8: estremi dichiarati; §17: finestre temporali specifiche | `InLiteralRange`, Bound inclusive/exclusive; §4.1.1 range non vuoto per incompatibilità dei parametri | Sintassi; tipo; comportamento; validazione | Parziale; incompatibile sui parametri invertiti e sui tipi non rappresentati | Non sostituire ogni range temporale RF con una finestra [inizio,fine); testo/Time/Year richiedono riscontro | 06 §§8,17; 07 §§2,4,5,18 | C07, C.4.5, X9 |
| C08 `is not between` | Complemento RF/JSON, SQL ordinary NOT BETWEEN | Negativi typed su valori noti; UNKNOWN preservato | Nessun costruttore dedicato; su dominio ordinabile noto ipotesi `Less(lower) OR Greater(upper)` | Struttura; comportamento | Parziale; non traduzione generale approvata | Possibile espansione in gruppi e diversa gestione missing/kind/inversione | 06 §§8,19,20; 07 §§4,14,16,18 | C08, X1/X9; catalogo chiuso §4 |
| C09 `contains` | Testo substring; array elementi; JSON uno dei due secondo kind runtime | §8: contains letterale testo oppure appartenenza esistenziale a insieme dichiarato | `ContainsText` oppure `DeclaredSetContainsLiteral` | Nomenclatura; struttura/tipo | Equivalente sul sottoinsieme noto Text/Array scalare; parziale nel complesso | Va preservato il soggetto del test; JSON dinamico non ha shape scalar/set unica; enum RF ammette contains | 06 §8; 07 §§2,4,18 | C09, `opContains`, tipi 07 |
| C10 `does not contain` | Negazione di substring/appartenenza; missing true Go/JSON | §20 ammette negazioni typed senza NOT generico, ma non definisce qui un complemento generale di ContainsText | La union §4.1 non contiene NotContainsText o negazione di DeclaredSetContains | **Disponibilità e comportamento** | **Incompatibile con una semplice rinomina**; manca una forma generale nella union attuale | Il filtro noto `name does not contain "z"` non ha una traduzione diretta già definita; non basta aggiungere generic NOT di propria iniziativa | 06 §§8,20,29; 07 §§4,18,19 | C10, X6; elenco completo ScalarPredicate |
| C11 `is one of` | Lista non vuota; JSON mista, compreso Null | §8 membership di valori confrontabili; §19 null distinto dal letterale | `InLiteralSet<Literal<T>>`, omogeneo e senza Null | Sintassi; tipo; comportamento | Equivalente su valori noti omogenei compatibili; parziale per catalogo intero | Liste JSON miste/null richiedono una decisione e forse più test distinti; dedup non è il problema principale | 06 §§8,19; 07 §§2,4,16,18 | C11, X5, test membership presenza |
| C12 `is not one of` | Complemento, con missing true Go/JSON | Diversità di valori noti; niente complement arbitrario | §4.1 cita espansione finita AND di `Different` come possibilità UI | Struttura; comportamento; limiti | Parziale, solo lista compatibile e sottoinsieme noto | Espansione di lista grande può cambiare numero di nodi/profondità; Null e kind misti non sono normali Different | 06 §§7.5,19,20; 07 §§4,6.4,16,18 | C12; nessun limite specifico alla lunghezza lista RF |
| C13 `starts with` | S0/J String, missing false | §8 prefisso letterale su testo | `StartsWithText` | Sintassi; tipo/ignoto | Equivalente su Text noto, parziale sul resto | I JSON non testuali non hanno già la stessa lettura typed; UNKNOWN non esposto dal RF | 06 §§8,19; 07 §§4,14,18 | C13; `opStartsWith` |
| C14 `ends with` | Come C13 per suffisso | §8 suffisso letterale | `EndsWithText` | Come C13 | Come C13 | Come C13 | Come C13 | C14; `opEndsWith` |
| C15 `is before` | D/Y, literal assoluto, strict; non JSON | §8 ordine temporale; §17 distingue istanti/date/calendario | `OrderedLiteral(Less)` per tipi ammessi; oppure test temporale di proprietà quando pertinente | Nomenclatura; tipo; normalizzazione temporale | Parziale | Date/DateTime possono avere forma corrispondente; Time e Year non sono varianti scalar autonome in §2.2 | 06 §§8,17; 07 §§2,4,5,16,18 | C15, C.4.4 |
| C16 `is on or before` | Come C15 inclusivo | Bordo incluso | `OrderedLiteral(LessOrEqual)` o forma temporale equivalente | Come C15 | Parziale | Come C15; preservare inclusione del bordo | Come C15 | C16 |
| C17 `is after` | Come C15, strict > | Bordo escluso | `OrderedLiteral(Greater)` o forma temporale equivalente | Come C15 | Parziale | Come C15 | Come C15 | C17 |
| C18 `is on or after` | Come C15 inclusivo | Bordo incluso | `OrderedLiteral(GreaterOrEqual)` o forma temporale equivalente | Come C15 | Parziale | Come C15 | Come C15 | C18 |
| C19 `is true` | Unario B/J, missing/null false | Valore Boolean noto; §19 normale confronto su ignoto UNKNOWN | `EqualLiteral(Boolean,true)` | Sintassi/payload; risultato | Equivalente su Boolean noto; parziale su ignoto/JSON | Cambio del risultato diagnostico; membership RF isolata positiva può restare uguale | 06 §§8,19; 07 §§2,4,14 | C19; `opIsTrue` |
| C20 `is false` | Unario B/J, missing/null false | Boolean false noto, non "non true" su dati ignoti | `EqualLiteral(Boolean,false)` | Come C19 | Come C19 | Non tradurre in NOT(true) che includa l'ignoto | Come C19 | C20; `opIsFalse` |
| C21 `is empty` | Missing, null, String/Array/Map vuoti; JSON anche Object vuoto | §19.2 testo vuoto distinto da missing/null; §8 distingue qualità | Solo `IsEmptyText`; nessun empty generale di contenitore in ScalarPredicate | **Comportamento e disponibilità** | **Incompatibile come singola rinomina** | Cambia membership su missing/null; array/map/object JSON non coperti dal solo test testuale | 06 §§8,19,20; 07 §§2,4,18 | C21, X3; `opIsEmpty` |
| C22 `is not empty` | Complemento del precedente | Valore noto non vuoto non è definito come complement universale del dato ignoto | Nessun costruttore dedicato; Different da String vuota è solo candidato per Text noto | Comportamento; disponibilità | Parziale per Text; manca equivalenza generale sui contenitori | Non sostituire con `HasKnownValue`, che ammette valori vuoti validi | Come C21 | C22, X3; closed union |
| C23 `is null` | Null presente in Go/JSON; SQL ordinary include missing | §19.2 null presente, distinto da missing | `IsExplicitNull` | Nomenclatura; comportamento executor | Equivalente Go/JSON osservabile; incompatibile con SQL ordinary che ha perso presenza | Non attestare una traduzione uniforme del nome senza contesto; nullable Object non compila | 06 §§8,19; 07 §§2,4,15,18 | C23, C.4.2/C.4.3, X4/X11 |
| C24 `is not null` | Missing true Go/JSON, SQL ordinary false | §8/19 distingue dato noto, missing e null | `HasKnownValue` **non è equivalente**; non c'è `Not(IsExplicitNull)` | Comportamento; forma | Incompatibile come rinomina | JSON missing perderebbe membership; eventuale combinazione di test richiede semantica/decisione esplicita | 06 §§8,19,20; 07 §§4,14,18 | C24, X1; dispatcher |
| C25 `exists` | Presenza Go/JSON; SQL ordinary non-NULL; Object per discendenti | Presenza anche con null in fotografia attendibile | `IsPresent` | Nomenclatura; rappresentazione/behavior | Equivalente Go/JSON osservabile; parziale SQL | Non confondere con `HasKnownValue`; rappresentazione di oggetti vuoti da chiarire | 06 §§8,19; 07 §§2,4,15,18 | C25, X4; object compiler |
| C26 `does not exist` | Assenza Go/JSON; SQL ordinary NULL; Object tutti discendenti assenti | Missing certo, non null | `IsMissing` | Nomenclatura; rappresentazione/behavior | Come C25 | Explicit null e empty Object possono essere riclassificati dal warehouse | Come C25 | C26, X4; reader e compiler |

Il problema non si limita a UNKNOWN: ci sono operatori negativi privi di una forma nella union, ordine lessicografico già ammesso, Time/Year non esplicitati, contenitori/JSON dinamici e liste Null miste. Non è dimostrata l'inclusione completa del record filtering nel solo catalogo progettato di ScalarPredicate.

### D.3 Contratto trasversale e funzionalità nuove

| ID / proprietà | Record filtering effettivo | Significato 06 | Rappresentazione 07 | Differenza | Compatibilità | Impatto sui filtri esistenti | Documenti potenzialmente interessati | Evidenze codice/test |
|---|---|---|---|---|---|---|---|---|
| R01 Soggetto e nome Filter | Un record: evento, utente importato, profilo | §§2,6,8: soggetto profilo corrente | §14 root su CurrentProfile, cinque primitive | Nomenclatura/contesto/disponibilità | Parziale: il root Unified non descrive tutti gli usi RF | Un solo core.Filter deve mantenere validi gli ingressi su eventi/import, senza assegnargli semantica di platea Profiles | 06 §§1,2,6,24,26; 07 §§1,14,15,18 | B.7; `Workspace.Events`, `ApplicationUsers`, collector |
| R02 Raw path e riferimento | Nome fisico puntato, risolto nello schema corrente; nessuna meaningRevision nel filtro | §§8,22,29.3: significato stabile, normalizzazioni dichiarate | §3 SemanticIdentity/ImmutableMeaningRevision; RawField tipizzato/registrato | Struttura e identità | Parziale, non mera rinomina | Binding futuro non può reinterpretare filtri salvati dopo rename/riassegnazione senza decisione | 06 §§22,29; 07 §§3,16,18,19 | `resolveFilterProperty`, `convertWhereToFilter`; B.8 ruoli |
| R03 Dato assente/null/empty | C.2/C.3 differiscono per tipo e executor | §19.2 stati distinti; missing/null UNKNOWN nei confronti | ReadResult e predicati distinti §2/4 | Comportamento | Incompatibile in casi dimostrati | Membership di negativi ed empty può cambiare; non estendere automaticamente l'eccezione a nuovi operatori | 06 §§8,19,20,28; 07 §§2,4,14,15,18 | C e X1–X5; fixture Go/JSON |
| R04 Risultati/composizione | Go bool; SQL usa anche SQL NULL interno; nessuna diagnostica UNKNOWN | §§19–20 ternarietà forte, soltanto MATCH include | §14.3 truth e failure separato | Comportamento/esecuzione | Parziale su sottoinsieme noto; incompatibile se si reinterpretano negativi | Stesso esito di membership non implica uguale evidenza; i body nuovi con quantificazioni non possono usare automaticamente bool=false per Unknown | 06 §§19,20; 07 §§14,15,18 | `Applies`; renderer; X1/X2 |
| R05 Filtro assente/gruppi/singleton | nil match-all; gruppi vuoti invalidi alla preparazione; singleton conservati | §20: match-all distinto da gruppo vuoto; singleton semanticamente identico alla foglia | §14 `AllProfiles`, singleton eliminato dalla forma canonica | Sintassi e struttura | Equivalente nel significato pubblico dei casi validi | DTO/persistenza/uguaglianza cambierebbero se si eliminassero gruppi o si serializzasse AllProfiles al loro posto | 06 §20; 07 §§14,16,18 | B.2; `Test_Filter_UnmarshalJSONLimits`, `Test_Where_Equal` |
| R06 Limiti | 4 livelli, 100 discendenti; nessun limite specifico lista membership | §7.5: root 3, body 2, scope 4, N 1–10, P 2–5 | §§6.4,14.2 indici di budget | Limite/struttura | Incompatibile come identica grammatica; riscrittura generale da verificare | Filtri RF validi a quattro livelli non ammissibili nello stesso albero target; espansioni negative consumano nodi | 06 §§7.5,29; 07 §§6.4,14,15,16,18,19 | `maxFilterDepth`, `maxFilterRuleCount`, X8 |
| R07 Canonical equality | Struttura/ordine sensibili; due meccanismi di change detection | §§20,29 significato preservato | §16 singleton/flatten/sort/dedup, normalizzazione literal | Struttura e operatività | Equivalenza logica sul sottoinsieme conforme; uguaglianza attuale diversa | Salvataggi, notifiche, aggiornamento collector e fingerprint possono cambiare anche a membership uguale | 07 §§16–18; 06 §§20,25,29 | B.6; `Where.Equal`, `Pipeline.changed`, X12 |
| R08 Precisione e soglie | Int64/UInt64 esatti; Decimal/JSON 76/37; Float arrotondato a bit size; soglie non limitate al range colonna | §10 confronti typed/unità; §8 nessuna coercizione; non fornisce un catalogo dei limiti RF | §2 Literal ExactValue/Number NumericDomain; §16 normalizzazione esatta | Tipo/limite; alcuni aspetti non definiti | Parziale; Float e domini numerici da verificare | Non restringere soglie a int8/decimal della colonna né convertire Decimal in float; l'esatta corrispondenza Float target va esplicitata | 06 §§8,10,22; 07 §§2,4,15,16,18 | DecodeFilterOperand, test widths/Decimal; C.4.8 |
| R09 Text/enum/identifier | Byte string, UUID/IP canonici; enum con contains; nessun dominio business Product/Order | §8 exact salvo normalizzazione catalogo; identifier dominio | §§2.3,3,4 operatori ristretti per tipo, nessun contains su enum/identifier | Tipo/comportamento | Parziale; alcune operazioni non previste | Non togliere ordine/contains solo per promuovere il vecchio String a un tipo semantico più ristretto | 06 §§8,22; 07 §§2–4,18 | preparazione String e `NormalizeUUID/IP`; C01/C03/C09 |
| R10 Quantità, money, duration | Semantiche già nello schema; confronti numerici nell'unità del campo, valuta anche opzionale | §10 unità/dominio/additività, valuta uniforme/dinamica esplicita | §§2.2/2.3,4.2 witness e TaggedMoneyField | Struttura/semantica nuova oltre metadata esistenti | Parziale | I letterali RF non hanno una seconda unità o valuta; non inventare conversione o rifiuto a posteriori per money senza currency | 06 §§8,10,22; 07 §§2,4.2,7,18 | tools/types.Semantics, schema Profiles; B.4 |
| R11 Tempo assoluto RF | D/Y letterali; Time/Year; offset API, UTC in persistenza, literal SQL senza offset; range inclusivo | §§8,17: Date/DateTime, istanti/offset/calendario dichiarati | §§2,4.3,5,16: bounds e normalizzazione | Tipo, sintassi, comportamento executor | Parziale; alcuni casi incompatibili | Query API e filtro persistito possono rendere diversamente l'offset; nessuna conversione universale a timeframe | 06 §§8,17; 07 §§2,4,5,16,18 | C.4.4/C.4.5, X9/X10 |
| R12 Unico T e snapshot | Nessun T semantico nel filtro RF; query/list/count sono operazioni; listener valuta arrivi | §§3,17–19: medesimo T, current vs historical distinto | §§5,14.3/14.4/15 T ambientale/evidence | Disponibilità/esecuzione nuova | Da verificare per nuova estensione | Nessun obbligo di trasformare ogni evento RF in una valutazione storica; count/listing esistenti non certificano uno snapshot condiviso | 06 §§3,17–19; 07 §§5,14,15 | `Workspace.Profiles`, PG/SF Query count/read; B.8 |
| R13 Capability del contesto | Filtro ammesso salvo Source/Database/User; niente livello Record/Unified | §§7,19.6,26 autorizzano forme e capability workspace, ma non i contesti operativi stabiliti dall'utente | §§6,14.5,15 validation shape/registry/capability | Disponibilità e validazione | Parziale: manca la nuova dimensione di ammissione | Le forme nuove devono essere rifiutate nei contesti RF anche via API; non ignorate. Il divieto database-source resta distinto | 06 §§1,2,7,19.6,26; 07 §§1,14.5,15,18 | `validatePipelineToSet:154`, chiamanti B.7 |
| R14 Destinazioni e popolazione | Where su Profiles prima della trasformazione; consensi/matching separati | §§2,24 membership dei profili; non definisce un export operativo | §14 membership su Profiles | Integrazione/disponibilità | Compatibile come punto di selezione; estensione da realizzare | Preservare selezione RF, aggiungere condizioni estese senza spostarle sul formato output o scavalcare consensi | 06 §§2,24,26; 07 §§1,14,15 | `exportProfiles:65`, `ProfileRecords:458`, Pipelines.md |
| R15 Persistenza/versioni | JSONB/state notifications e schema corrente; nessuna versione filtro | §29 stabilità del significato; nessuna migrazione RF concreta | §19 meaningRevision; wire/backward compatibility rimandati | Struttura/compatibilità | Da verificare sui dati; problema già noto per typed values | Non caricare vecchie String JSON numeriche cambiandone silenziosamente membership; futuro AST è un secondo problema | 06 §29; 07 §§3,18,19; audit 10 §9 | `MarshalWhere`, loader, RELEASE-UPGRADE; X13 |
| R16 Validazione in fasi | Decoder struttura, prepare schema, loader differente, renderer JSON difensivo; errors API | §§19.1/19.7 distinguono invalidità, indisponibilità, failure | §14.5 StaticAssessment; §15 livelli; nessun protocollo Fase 4B definito | Struttura/disponibilità | Parziale | Conservare i rifiuti API; non degradare invalidità/errore in UNKNOWN. Il loader non è già la validazione completa dei nuovi vincoli | 06 §19; 07 §§14.5,15 | B.2/B.5; test persistence validation/compiler errors |
| R17 Confronti fra proprietà | Values solo letterali, nessun RHS property reference | §§8,9.2,10.2 confronti profilo–profilo ed evento–profilo corrente | §4.1 Equal/OrderedCurrentProfile; §7 Direct | **Funzionalità nuova**, non semplice rinomina | Parziale disponibilità: attualmente assente | Non reinterpretare una String RHS come un path; cambierebbe il significato di filtri salvati | 06 §§8–10; 07 §§4,7,18 | `FilterCondition.Values`, X14. La frase di 07 §18 su un confronto legacy fra proprietà non descrive un supporto di questa baseline |
| R18 V riduzioni, E, K, P, T | Nessuna rule corrispondente; solo proprietà del singolo record | §§9–14 cinque primitive e contesti | §§7–9,12–14 union chiuse, owner/guardie | **Funzionalità nuove** | Da verificare nell'estensione; non difetto RF | Nessun obbligo di emulare storia con filtri riga; il riuso dei predicati non autorizza qualsiasi contesto | 06 §§6–14; 07 §§6–9,12–15 | FilterRule limitato a gruppo/condizione; B.7 |
| R19 Ruoli/identità/evidence | Schema typed e semantiche; ruoli correnti solo nel branch previsto; nessuna certificazione dei nuovi scope | §§9,12,15,16,22 contratti mirati | §§3,10,11,14.4 definizioni versionate | Disponibilità nuova/integrazione | Parziale | Riutilizzo dei ruoli di profilo da discutere; non chiamare business identity un path email, KPID o un nome campo | 06 §§9,12,15,16,22; 07 §§3,10,11,14,15 | B.4/B.8; `ProfileRoleAssignments` al commit distinto |
| R20 Authoring Profiles | API RF già presente, pagina baseline sempre null; redesign non incluso | §§5,21,25 disclosure/recipe/diagnostica | §17 presentation/provenance separate | Disponibilità e presentazione | Parziale | Il nuovo editor dovrà conservare kind/precisione e limitare capability; cronologia UI non è semantica o catalogo persistito | 06 §§5,21,25; 07 §§16,17 | `useProfiles:124` baseline; B.8 branch pinned |

R17–R20 non chiedono di cambiare il significato di una foglia RF: evidenziano lavoro nuovo o integrazioni che non sono ancora disponibili. I vincoli di owner, grana, selezione ed evidenza non diventano divergenze del record filtering soltanto perché non esistevano nel suo perimetro.

### D.4 Controesempi minimi e limiti delle deduzioni

Per brevità una condizione è scritta `path operatore [valori]`; deve essere contenuta in una root AND singleton per il JSON pubblico. Gli schemi rendono gli operatori ammessi, salvo dove l'esempio riguarda proprio un rifiuto. "Target" significa comportamento progettato prima della riconciliazione, non decisione approvata di sostituire quello esistente.

| ID | Dati/schema e filtro | Risultato RF effettivo | Documento 06/07 | Conseguenza verificabile |
|---|---|---|---|---|
| X1 Negativo JSON su missing | `properties={}` JSON; `properties.x is not [100]` | Go e SQL JSON true; profilo incluso | 06 §19.2 normale diversità su proprietà assente → UNKNOWN, escluso | Incompatibilità di membership, non solo etichetta del risultato. Anche `is not null` sullo stesso missing è true RF, mentre HasKnownValue non lo è. |
| X2 Negativo ordinary | Record `{}`, `n` Int ReadOptional; `n is not [100]` | Go true; SQL ordinary su colonna NULL produce SQL-U, escluso | 06 → UNKNOWN, escluso | Contratto RF già diverso fra executor; decidere la riconciliazione non autorizza a dichiarare Go o SQL universalmente corretto. |
| X3 Empty | `properties={}` JSON, oppure `properties={"x":null}`; `properties.x is empty` | true per entrambi. Anche `x:[]`, `x:{}`, `x:""` danno true | 06 §19.2 IsEmptyText non corrisponde a missing/null; solo testo vuoto dà MATCH | Sostituire con IsEmptyText elimina membri; contenitori richiedono significato ulteriore. |
| X4 Presenza/null | Schema ordinary `x` String Nullable e ReadOptional; record `{x:nil}`; `x exists` | Go true; una colonna SQL NULL risulta non esistente. Con record `{}`, `x is null` è false Go ma SQL true | Fotografia attendibile distingue missing da null; IsPresent include null, IsExplicitNull esclude missing | Informazione di presenza persa in SQL ordinary; l'omonimo test JSON la preserva. Non usare lo schema Profiles non-nullable per fingere che l'operatore null ordinary vi sia ammesso. |
| X5 Null in lista | `properties={"x":null}`; `properties.x is one of [null]` | true; con path assente false | Null non è un `Literal<T>` di InLiteralSet in 07 §4; IsExplicitNull è distinto | La forma diretta è invalida nel target; possibili combinazioni di predicati sono ipotesi, non mapping già approvato. |
| X6 Negativo testuale noto | `name="abc"` String; `name does not contain ["z"]` | Go true; renderer ordinary produce il relativo predicato negativo | ScalarPredicate §4.1 ha ContainsText ma non il suo negativo e non un NOT generico | Manca una forma generale corrispondente, anche senza dati mancanti. |
| X7 Ordine su testo | `name="Anna"` String senza enum; `name is less than ["M"]` | Go true per ordine lessicografico | 06 §8 non include testo fra i tipi ordinati; 07 §4.1.1 non dà l'istanza a testo libero | Restrizione di capacità, oltre al problema delle collazioni SQL. |
| X8 Profondità | `AND(a,OR(b,AND(c,OR(d,e))))`, dove a–e sono condizioni valide | 4 livelli di gruppi, 8 discendenti: valido RF | Root CanonicalBool<3>; stessa struttura non ammessa | Singleton/flatten non bastano perché i gruppi sono alternati. Non dimostra impossibilità di qualunque riscrittura logica, ma questa non è già la policy approvata e può espandere l'albero. |
| X9 Range Date invertito | `d=2026-10-20`; `d is between ["2026-10-20","2026-10-10"]` | Go true perché coincide con un estremo; SQL BETWEEN false | Range staticamente non vuoto/compatibile richiesto da 07 §4.1.1; nessuna concessione documentata per questo caso | Incompatibilità di validazione e divergenza executor preesistente, dedotta dal codice. Nessun test mirato nuovo eseguito. |
| X10 Offset DateTime | Stesso istante soglia scritto `2026-10-24T12:00:00+02:00` o `2026-10-24T10:00:00Z` | Go confronta lo stesso istante. I renderer producono literal ore 12 oppure 10; persistenza riscrive UTC | 06 §17 e 07 §16 richiedono istanti coerenti | Divergenza del rendering dimostrata dai test; eventuale differenza di membership va verificata sullo storage. Con dati UTC senza timezone `10:30`, il confronto > delle due forme può divergere. |
| X11 Nullable Object | Schema `o` Object Nullable/ReadOptional con figlio `a`; record `{o:nil}`; `o is null` | Preparazione ammessa, Go true; compiler delle colonne `o_a` restituisce errore | IsExplicitNull è un test distinto di proprietà osservabile | Mismatch ammissione/esecuzione generale, non un nuovo filtro né un esempio valido dello schema Profiles corrente. |
| X12 Uguaglianza | `AND(a,b)` e `AND(b,a)`, con dati noti e stesse foglie | Membership uguale; `Where.Equal` false e serializzazioni diverse | 07 §16 canonical sort: uguaglianza canonica | Cambia il rilevamento delle modifiche anche senza cambiare risultati; serve distinguere identità di editing, forma persistita e semantica. |
| X13 Dati precedenti a typed values | JSON x Number 100; filtro persistito precedente `Is` con `values:["100"]` | Al genitore **82e7be3a…** il loader crea String più possibile Number e il confronto è true; a **e66eea846…** quella String vale solo testo, confronto false | 06/07 non forniscono una migrazione dei filtri installati | Problema di rilascio già esistente, separato dalla riconciliazione futura; la leggibilità del JSON non prova preservazione. |
| X14 Nuovo confronto fra proprietà | `{preferred:"it", communications:"en"}`; richiesta `preferred diverso da communications` | Il DTO non ha RHS property; String `"communications"` è soltanto un letterale | 06 §8 e 07 EqualCurrentProfile la esprimono | Funzionalità nuova; mai reinterpretare vecchi letterali come riferimenti. |
| X15 Array temporale Snowflake | Schema `dates` Array(Date); record con elemento 2026-10-20; `dates contains ["2026-10-20"]` | Go true; PG usa la data come letterale per ANY; SF rende `CONTAINS("DATES", '')` passando al serializer il tipo Array | 06 appartenenza esistenziale allo stesso valore; 07 DeclaredSetContainsLiteral avrebbe un literal Date | Divergenza di rendering preesistente e non fedeltà del letterale, dedotte dal codice senza provare l'esito live della query SF. |

Evidenze specifiche per X13, verificate in lettura in questa attività: `git show 82e7be3a1939340cfc3374cf9f8458a63f21a43b:core/internal/state/where.go`, `JSONConditionValue` alle linee 218–226, marshaling 306–308 e parsing 380–388; `core/internal/filters/applies.go:106–123` allo stesso commit. Sono fonti del **genitore precedente**, non comportamenti attribuiti a HEAD. Gli altri controesempi rinviano ai rami e test delle sezioni B/C; non costituiscono l'esito di nuove prove end-to-end non eseguite.

## E. Decisioni da prendere

L'ordine indica impatto e dipendenze. Le domande sotto sono decisioni di riconciliazione da discutere; non richieste di approvazione immediata a un'implementazione, né soluzioni già scelte.

1. **Confine del significato RF dentro Unified.** Stabilire esplicitamente come si preservano le condizioni RF già ammesse quando vengono usate per interrogare/esportare profili insieme a condizioni nuove. I casi C02/C08/C10/C12/C21/C24 hanno differenze dimostrate rispetto al progetto. Il requisito di inclusione funzionale è già assunto; una traduzione globale alle attuali regole 06 non è ancora dimostrata. Dipendenze: nessuna; condiziona tutte le successive. Fonti R01/R03/R04, X1–X5.

2. **Contratto per le divergenze già interne al RF.** Identificare cosa preservare nei diversi ingressi, distinguendo intenzione documentata JSON da ordinary bool/SQL NULL, presenza/null, Object appiattiti, range temporali invertiti e rendering degli offset. Un'eventuale correzione di questi casi cambia comportamento e richiede motivazione/approvazione propria; non può entrare come effetto collaterale della Fase 4B. Per Array ordinary Snowflake il literal temporale vuoto è già deducibile dal rendering; la validità della funzione SQL e la membership richiedono ancora una prova runtime. Nessuna correzione è già autorizzata. Dipende da 1 per l'estensione, ma l'accertamento dell'esistente è indipendente. Fonti C.4, X2/X4/X9–X11.

3. **Completezza del catalogo condiviso.** Decidere il trattamento di `does not contain`, `is not between`, `is not one of`, `is not empty`, `is not null`, dei test sui contenitori e del testo ordinabile. Alcune forme possono avere espansioni sul sottoinsieme noto; altre non hanno un equivalente generale nella union progettata. Un NOT generico o un nuovo predicato non è una decisione già presa. Dipende da 1–2, prima di dichiarare che ogni RF è rappresentabile. Fonti D.2, X6/X7.

4. **Missing/null/empty/known value e JSON dinamico.** Stabilire la corrispondenza dei test esistenti, compresa la lista mista String/Number/Boolean/Null e il significato di una proprietà JSON che può contenere String, Number, Array o Object. La distinzione proposta da 06 può restare valida per nuovi test, ma non giustifica rinominare RF `is empty` in IsEmptyText o `is not null` in HasKnownValue. Dipende da 1–3. Fonti C.2/C.3, X1/X3/X5.

5. **Ammissibilità strutturale e budget.** Riconciliare i 4 livelli/100 discendenti RF con root 3/body 2 del progetto, i costi delle eventuali espansioni e il limite oggi non specificato delle liste. Decidere cosa deve essere accettato in authoring, traduzione e caricamento, senza usare una semplificazione per cancellare un'invalidità né rifiutare in silenzio filtri salvati. Non è stata dimostrata una riduzione generale compatibile con tutti i limiti. Dipende da 3–4. Fonti R06, X8.

6. **Domini e rappresentazioni dei valori condivisi.** Chiarire il mapping di Int/UInt64, Float approssimato, Decimal/JSON 76/37, Time/Year, enum, UUID/IP e semantiche di unità/valuta già disponibili. La precisione delle soglie RF non segue quella del valore memorizzabile nella colonna. Il modello ExactValue non attesta ancora da solo il trattamento dei Float esistenti. Non cambiare normalizzazioni o validazione degli operandi come conseguenza di una nuova etichetta semantica. Dipende da 3–4; la parte temporale anche da 2. Fonti R08–R11, B.4/C.4.

7. **Persistenza installata e migrazioni distinte.** Prima del rilascio dei typed values serve la decisione già richiesta da RELEASE-UPGRADE sui precedenti operandi JSON. Separatamente, stabilire come l'estensione di core.Filter mantenga leggibilità, significato e notifiche dei filtri typed esistenti. Non è richiesta qui una scelta di formato/versioning o un updater. Le due compatibilità non vanno fuse in un'unica conversione presunta lossless. Dipende dalle decisioni sul significato, non dall'adozione preventiva di un nuovo AST pubblico. Fonti R15, X13.

8. **Uguaglianza, normalizzazione e identità del documento.** Decidere a quali rappresentazioni si applichino sort/flatten/dedup, quale uguaglianza usino salvataggio/collector/cache e come resti ricostruibile l'editing. Il requisito del Canonical AST interno non implica eliminare gruppi/ordine nel DTO RF o cambiare tutti i fingerprint. La trasformazione deve preservare semantica e limiti approvati. Dipende da 3–7. Fonti R05/R07/R20, X12.

9. **Ruoli esistenti e stabilità dei riferimenti.** Definire la relazione fra path RF, assegnazioni mutabili di profile-roles e significati versionati, senza duplicare né equiparare automaticamente quei concetti. È una decisione d'integrazione necessaria per non reinterpretare filtri dopo riassegnazioni. I contratti di azione/identità/grana/evidence sono nuove dipendenze, non già risolte da profile-roles. Dipende da 6–7; verificare sul risultato del branch approvato. Fonti R02/R19, B.8.

10. **Validazione delle capability operative nei punti effettivi.** La scelta di prodotto è già fissata: Unified solo Profiles API/Admin ed export profili app/file/database. Resta da precisare l'applicazione server alle forme condivise/estese, alle anteprime e ai filtri caricati. Distinguere il livello operativo dalla validità di shape, dalle capability del workspace e dalla sufficienza delle evidenze. Nessun cambiamento al divieto RF Source/Database/User è stato approvato. Questa definizione deve usare i repertori riconciliati di 1–6; non richiede di inventare ora il formato pubblico completo. Fonti R13/R16, B.7.

11. **Coerenza di valutazione nelle superfici Unified.** Il T comune del modello comportamentale rimane un requisito significativo; non è garantito dai count/listing RF o dagli executionID del redesign. Va chiarito il contratto di osservazione della stessa richiesta nelle superfici autorizzate, incluse spiegazione e UNKNOWN, senza introdurre qui un protocollo esecutivo. Dipende da 1/9/10 e va discusso prima della progettazione esecutiva 4B. Fonti R12/R14/R20.

Le decisioni 1–8 riguardano direttamente contratti o compatibilità già osservabili. Le 9–11 circoscrivono le integrazioni necessarie alla decisione dell'utente. Nessuna di esse approva `legacyMode`, policy configurabili di UNKNOWN, alias impliciti o predicati opachi.

## F. Parti delle specifiche potenzialmente interessate

La tabella localizza gli interventi che **potrebbero** seguire alle decisioni; non li esegue né presume che debba cambiare il record filtering. Se una decisione conserva il RF, può essere la specifica progettuale a dover restringere o precisare la propria applicabilità.

### F.1 Documento 06

| Sezioni precise | Motivo della possibile revisione | Parti non automaticamente da cambiare |
|---|---|---|
| §§1.1–1.2, 2, 6; §30 e dichiarazione di freeze come indice delle decisioni | Esplicitare rapporto fra contratto RF preesistente, Unified e un solo core.Filter; il freeze precedente non risolve il conflitto | Le cinque primitive come categorie delle capacità comportamentali non vengono smentite da questa verifica. |
| §7.5, §29.1, §29.4 | 4 livelli RF/100 nodi/liste rispetto a 3/2 livelli; eventuali espansioni; perimetro delle regole di ammissione | Budget di scope, N 1–10 e P 2–5 riguardano nuove forme; non emerge una ragione RF per alterarli. |
| §8, §§9.2–9.3 per i predicati condivisi | Catalogo effettivo dei 26 operatori, JSON dinamico, contenitori, ordine testuale, enum, literal e confronti con proprietà | Same-occurrence resta un requisito distinto dal normale AND su un record; non ne viene proposto il cambiamento. |
| §§10.1–10.3 limitatamente ai tipi/operandi condivisi | Precisione soglie, Float, Time/Year, unità e money già presenti; nuovo confronto fra proprietà distinto dai letterali | Guardie, distinzione ratio/share e atomicità delle riduzioni non sono incompatibilità del RF. |
| §§17.1–17.3, §28.12 per i casi temporali condivisi | Normalizzazione degli istanti, bounds RF inclusivi, range invertiti e tipi temporali; separare assoluti RF e periodi comportamentali | T unico per una valutazione comportamentale, rolling vs calendario e ancore lessicali restano concetti validi del nuovo perimetro. |
| §§19.1–19.2, 19.6–19.8, §§20–20.1 | Missing/null/empty, complementi esistenti, verità SQL e bool, invalidità/failure/capability | UNKNOWN per evidenza storica incompleta, deadline pendente e ordine insufficiente è un requisito nuovo, non confutato dal RF. La sua applicabilità alle foglie condivise va decisa. |
| §§22.1–22.3, §29.3 | Path fisici, semantiche già disponibili, profile roles previsto, significati e revisioni | Ruolo di profilo, business identity e contratto autorevole restano nozioni distinte. |
| §§5,21,25; §§26.1/26.5 | Authoring condiviso, capability per contesto, anteprima/count, distinzione editor e documento semantico | Recipe come presentazione/espansione senza cambiare significato resta coerente; il redesign non ne implementa già l'intero contratto. |
| §28, in particolare §§28.1,28.18, più nuove regressioni RF da concordare | Acceptance deve comprendere i casi condivisi e gli impatti approvati, non attestare soltanto i nuovi intenti | Non serve riclassificare o eliminare requisiti business del benchmark per adattare una rappresentazione tecnica. |

### F.2 Documento 07 e modello finale 08

| Sezioni precise 07 | Motivo della possibile revisione | Riepilogo finale da tenere coerente dopo un'eventuale decisione |
|---|---|---|
| §§1,1.1 | Assunzioni senza catalogo RF e portata delle invarianti sul solo Unified o sulle parti condivise | F1, F8; documento 08 corrispondente. |
| §§2.2–2.3 | Raw JSON, Scalar/DeclaredSet vs contenitori correnti; Int/Float/Decimal, Time/Year, literal Null in membership, soglie/unità | F2. |
| §§3–3.2 | Relazione path/schema/versione e ruoli esistenti, senza equivalenza automatica con meaningRevision | F2, F5, F8. |
| §§4.1–4.1.1 | Corrispondenza completa dei 26 operatori, negativi mancanti, ordine testuale, empty/null, liste e restrizioni enum | F2, F6. |
| §4.2, §7.1 limitatamente a valori/property tests | Semantiche money esistenti, letture correnti e nuovo RHS property; distinguere capacità nuova da traduzione RF | F2, F3-V. |
| §§4.3,5.1–5.3 | Bound assoluti e temporali RF, Time/Year e normalizzazione; nessuna sostituzione automatica con finestre relative | F2. |
| §6.4, §§14.1–14.2 | Profondità/count, root obbligatoriamente gruppo RF vs Leaf/AllProfiles canonici; portata del root su profilo | F1, F4. |
| §§14.3–14.5,15 | Bool RF/SQL-U vs semantic UNKNOWN; validazione nei contesti operativi oltre a shape/registry/workspace | F1, F6. |
| §§16.1–16.3,17 | Uguaglianza strutturale esistente, change detection, forma editoriale/DTO/persistenza, ordine e identità dei nodi | F7; presentazione/provenance. |
| **§18 completo** | Ora il catalogo reale è conosciuto: sostituire le assunzioni con compatibilità dimostrate e casi aperti. La menzione di un confronto legacy fra proprietà non descrive la baseline | F8 e dichiarazioni di losslessness. |
| §19, §§23/23.1/23.2 | Migrazione typed values separata, evoluzione di limiti/tipi, aggiornamento delle attestazioni di copertura e dei limiti dichiarati | F8; rinvii a F2/F4/F6/F7. |

07 non definisce un wire format pubblico completo né obbliga a usare la sua ADT testuale come struttura Go esposta: §§17 e19 rinviano tali aspetti. L'estensione di `core.Filter` è il vincolo approvato dall'utente; non si deduce da questa verifica una scelta di compiler, package o nuovo formato da implementare.

### F.3 Decisioni progettuali presumibilmente non coinvolte

Non emergono incompatibilità RF che, da sole, richiedano di cambiare:

- il confine filtro/analytics/workflow, l'assenza di SQL o formule generiche nel filtro e le esclusioni di join fra profili: 06 §§23–24,27; 07 §20;
- separazione base → selezione → verifica, nessun fallback a un'altra esperienza e differenza fra last e last qualifying: 06 §11; 07 §8;
- grana/identità, non-vacuità di All, differenze fra None/All/Share e preservazione degli incerti nei denominatori: 06 §12; 07 §9;
- ordine affidabile, passi positivi distinti, blocco di correlazione e assenze locali: 06 §13; 07 §12;
- catalogo dei pattern e qualified-period cardinality: 06 §14; 07 §13;
- separazione dei contratti autorevoli e delle forme di derived evidence, con consumer autorizzati: 06 §§15–16; 07 §§10–11;
- distinzione current/event-captured/historical/authoritative e divieto di ricostruire il passato da dati correnti: 06 §18; 07 §§3,10,11;
- separazione di failure tecnico e risultato semantico e impossibilità di inventare evidenza mancante: 06 §§19.1/19.7; 07 §§14.3–14.5.

"Non coinvolte" riguarda questi principi delle **nuove capacità**. I predicati condivisi usati nei loro where/body potranno essere interessati dalle decisioni su tipi, negativi e UNKNOWN: non si certifica qui l'immutabilità di ogni firma o esempio di quelle sezioni.

Il documento 03 resta la fonte dei bisogni business. Il documento 08 richiederebbe un aggiornamento derivato soltanto dopo le decisioni su 07. L'audit 10 può restare una fotografia storica: le sue conclusioni generali sui tipi ordinari sono integrate dai rilievi C.3/C.4 di questo rapporto, senza modificare ora alcuna fonte.

## G. Incertezze e verifiche mancanti

### G.1 Cosa non è determinabile dai test locali eseguiti

| Verifica mancante | Cosa sappiamo già | Cosa rimane aperto e perché conta |
|---|---|---|
| Matrice differenziale completa ordinary Go/PG/SF | Dispatch Go e SQL prodotti sono stati letti; i test locali passano | Non attestata la membership per tutte le combinazioni ordinary, tipi temporali, Array e valori limite. Le differenze NULL dedotte dal SQL non costituiscono una nuova esecuzione su dati reali. |
| Snowflake reale | Renderer e test di rendering presenti; credenziali test assenti | Array ordinary `CONTAINS` e relativo caso temporale X15, numeric storage, collazioni e runtime devono essere provati nell'ambiente appropriato. Non è stata avviata una suite che possa scrivere su un servizio condiviso. |
| Offset e precisione temporale end-to-end | Test fissano literal senza offset, PG microsecondi/SF nanosecondi; persistenza UTC | Effetto su query dirette, filtri caricati e dati memorizzati con timezone/precisione effettive. X10 espone una condizione riproducibile, non una prova live completata. |
| Range invertiti | Differenza dedotta dal ramo time.Time e dal normale BETWEEN SQL | Nessun test mirato individuato sui bordi invertiti; comportamento desiderato non documentato. Il test di compiler Object esercita `Is`, non specificamente il caso nullable `IsNull` di X11. |
| Validazione dati malformati | Reader normali validano/normalizzano; evaluator interno può panic o false | Non esiste un risultato semantico uniforme già disponibile da adottare come UNKNOWN. Serve distinguere invarianti interne da evidenza proveniente da storage controllato dall'utente. |
| Float non-finiti e domini NumericDomain | Dati non-Real supportano NaN/Infinity; le soglie RF sono finite e arrotondate a bit size | Il target non precisa in modo sufficiente la corrispondenza di queste rappresentazioni e non è stata eseguita una matrice differenziale sui warehouse. Non si dichiara ogni Float RF già incompatibile con Number. |
| Limiti complessivi HTTP | Il filtro impone 4/100, 80 rune e limiti numerici; membership non ha un massimo proprio individuato | Non attestato in questo incarico un limite di body/query string uniforme per tutti gli ingressi/deployment. Non inventare un nuovo massimo come se fosse contratto attuale. |

### G.2 Informazioni esterne al codice necessarie per le decisioni

1. **Filtri realmente installati**: distribuzione di profondità, operatori, tipi, cardinalità delle liste e uso dei negativi con campi assenti; client che inviano ancora operandi testuali precedenti ai typed values. Nessun ambiente o database di produzione è stato interrogato.
2. **Significato dei vecchi operandi JSON**: la precedente ambivalenza String/Number non rivela quale interpretazione fosse voluta in ogni filtro. La migrazione non è deducibile universalmente dal testo `"100"`.
3. **Contratto RF voluto nelle divergenze**: codice e test mostrano ciò che succede, ma non attestano approvazione funzionale di ogni caso. In particolare mancano una scelta generale sui negativi ordinary fra Go/SQL e una motivazione del rendering senza offset; un test di regressione non sostituisce tale decisione.
4. **Binding semantici effettivi**: cataloghi di domini, unità confrontabili, grane, deduplication, osservabilità, ordine e contratti autorevoli per i workspace destinatari. I nomi raw o i ruoli di profilo non forniscono da soli queste prove.
5. **Esito delle revisioni dei branch**: si conoscono soltanto i commit locali pinned; nessun merge è stato prodotto. Devono essere rivalutati authoring, count/list, kind/precisione, dipendenze schema e validazione di contesto sul codice realmente integrato. Il porting del redesign è già previsto dall'utente.
6. **Decisioni di prodotto sulla riconciliazione**: sono da approvare prima di modificare 06/07. Né l'etichetta frozen né l'esistenza di un comportamento corrente risolvono automaticamente il conflitto o ne autorizzano l'estensione ai nuovi operatori.

### G.3 Verificabilità e confine della consegna

I riferimenti locali al codice valgono per `e66eea846…`; i riferimenti di branch e del genitore sono esplicitamente distinti. La [directory di lavoro del rapporto](/private/tmp/krenalis-filtering-reconciliation-20261025) contiene il Markdown, le copie leggibili delle fonti, l'elenco degli hash dei documenti e il log dei test locali; i file temporanei non sono modifiche al repository.

Sono state completate la ricostruzione del contratto RF, le matrici dei 26 operatori e dei tipi, la distinzione fra executor e intenzione documentata, il confronto con 06/07 e l'elenco delle decisioni. Restano intenzionalmente aperte le scelte semantiche/architetturali e le verifiche esterne indicate. Nessun filtro nuovo, formato pubblico completo o attività della Fase 4B è stato implementato. Il controllo finale ha confermato HEAD e stato Git invariati, nessuna differenza tracciata e gli stessi hash SHA-256 per tutti i dieci documenti originali.
