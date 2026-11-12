# Catalogo eventi Krenalis

Il catalogo compilato contiene le 102 definizioni dei pacchetti consegnati.
È predefinito, condiviso fra workspace e disponibile all'avvio, senza caricamenti
di file o dipendenze Python a runtime.

## Fonti e rigenerazione

- `cdp-event-taxonomy.zip`, versione 1.1.0: ID, nomi tecnici, etichette e
  descrizioni originali in italiano, matcher. Conserva anche i metadati di
  ricerca e i 16 binding di compatibilità, che questo incremento non abilita.
- `event_texts_en.json`: traduzioni inglesi di `display_name` e `description`,
  indicizzate per ID, usate per i testi del catalogo compilato. Gli archivi
  originali e i nomi tecnici rimangono invariati.
- `cdp-event-properties.zip`, versione 1.0.0: il suo
  `cdp-event-properties.json` è la fonte delle proprietà e delle descrizioni.
  I JSON Schema dei payload sono derivati, non una seconda fonte da modificare.
- `SPECIFICA_EVENTI_CDP_PER_CODEX.md`: significato del catalogo originale,
  compresi requisiti futuri di personalizzazione e versionamento.
- `risposta-ricerca-proprieta-ambigue.md`: ricerca supplementare del 5 novembre
  2026, da distinguere dai contratti delle sorgenti e dalle scelte Krenalis.

Per rigenerare il file Go dal repository:

```sh
python3 filtering/event-catalog/generate_catalog.py
python3 filtering/event-catalog/generate_catalog.py --check
python3 -m unittest discover -s filtering/event-catalog -p 'test_*.py'
```

Il generatore verifica i manifest SHA-256 dei due archivi, la loro baseline
comune, la corrispondenza dei 102 ID e la completezza dei testi inglesi.
Riutilizza dal pacchetto delle proprietà i criteri del profilo `documented` e
l'espansione dei campi condivisi. Non modifica gli archivi, i JSON Schema
consegnati o altri file derivati dei pacchetti.
Il manifest verifica la coerenza interna, non l'autenticità: archivi e helper
eseguibili sono input del repository che devono già essere considerati fidati.

`native_type_choices.py` contiene le scelte non predefinite per coppia
`(event_id, path)`, insieme ai kind sorgente attesi. Non si ricavano semantiche
da nomi o sottostringhe delle descrizioni. Un'unione di tipi non censita o un
cambio dei kind di una scelta già censita interrompe la generazione.
`--check` verifica la corrispondenza byte per byte senza modificare il file Go;
il controllo viene eseguito anche dai test Go quando Python 3 è disponibile.

Il risultato è `core/internal/state/event_catalog_data.go`. Le dichiarazioni Go
contengono direttamente `types.Type`; nessun generatore viene eseguito all'avvio.
Gli schemi identici possono essere condivisi perché `types.Type` è immutabile.

## Rappresentazione attuale

`state.EventDefinition` espone i campi già concordati: ID, Type, Event, Name,
DisplayName, Description, Trigger e Schemas. `Trigger` rimane vuoto quando le
fonti non forniscono un campo separato: non viene inventata una garanzia di
emissione dell'evento. DisplayName e Description sono in inglese, come le
descrizioni delle proprietà già fornite dall'archivio delle proprietà.
Gli altri metadati del catalogo originale restano negli archivi; non sono stati
aggiunti al contratto runtime di questo incremento.

`Workspace.EventDefinitions` restituisce una copia dello slice.
`Workspace.LookupEventDefinition` confronta esattamente tipo e nome tecnico;
per page e screen usa la definizione generica quando non esiste una definizione
specifica per quel nome. Non cerca sinonimi né nomi di compatibilità.

La voce `legacy.identity_merge` è conservata come riferimento storico. La sua
presenza nel catalogo non abilita il tipo `merge` nel decoder. Analogamente,
la presenza di una definizione non prova che il workspace abbia ricevuto eventi
corrispondenti e non abilita nuove capacità del motore Unified.

Non vengono introdotti configurazione per workspace, versionamento, preset,
ruoli semantici, JSON Schema in ingresso, endpoint HTTP o tool MCP.

## Scelte dei tipi nativi

Lo schema nativo è una scelta Krenalis basata sul significato dei campi. Non è
una traduzione senza perdita di tutti i JSON Schema dei provider.

| Caso | Tipo nativo |
|---|---|
| Importi, prezzi, ricavi e riduzioni monetarie documentate | Decimal(18,4), Money senza valuta fissa |
| Quantità commerciali documentate | Decimal(18,4), senza unità o segno dedotti |
| Durate documentate | Float(64) reale, Duration in secondi |
| Offset video | Float(64) reale; Measurement/Second attende l'integrazione separata |
| Posizioni ordinali e conteggi documentati | Int(32) |
| Altri Number documentati | Float(64) reale |
| Rating | String, senza scala numerica implicita |
| Identificatore company.id e employees dei gruppi | String; nessuna conversione numerica implicita |
| Valore generico Number/String delle azioni | String; i valori specificamente monetari mantengono Money |
| Query prodotti String/Object e ad_type String/Array | JSON |
| Birthday documentato come data logica | Date |
| createdAt/created_at dei contratti identity | DateTime |
| Indirizzi email e URL documentati | String con la rispettiva semantica |
| Tipi ignoti, soltanto esemplificati, proposti o specifici di una destinazione | JSON |

Precisione, scala e dimensioni degli interi nella tabella sono decisioni locali,
non vincoli attribuiti a Segment o RudderStack. Per i telefoni si conserva String:
il testo fornito non è necessariamente nel formato canonico richiesto da Phone,
e la conversione è rinviata. Non viene imposta una valuta EUR o un fallback USD.

Gli oggetti sono aperti. Gli array di oggetti mantengono gli schemi degli elementi;
gli oggetti senza struttura dichiarata usano Map(JSON), gli array senza contratto
degli elementi usano Array(JSON). I campi sono facoltativi e diventano nullable
quando il profilo documentato include esplicitamente null. Non si applicano
enum o vincoli desunti dagli esempi.

Le chiavi `properties["Feed Type"]` della schermata e `traits["total billed"]`
del gruppo sono esempi fuori dalla grammatica dei nomi di `types.Property`.
Restano ammesse e preservate come campi JSON non dichiarati; non vengono
rinominate o presentate come proprietà native. Il package `types` è invariato.

Il vincolo canonico delle semantiche rimane quello del profile schema, con le
due eccezioni concordate per gli eventi: array di oggetti e proprietà nullable.
Il passaggio separato di Measurement a Float64 e l'aggiunta dei secondi alla
sua unità di misura non sono anticipati da questo incremento.

## Decoder e pipeline

Il decoder applica lo schema al contenuto normalizzato di traits e properties.
Gli altri campi dell'envelope continuano a seguire il decoder comune: i pacchetti
non introducono nuovi fallback, alias o interpretazioni delle sorgenti.
Per group viene usato lo schema dei tratti del gruppo, non quello della persona.
Gli eventi già persistiti non vengono rivalidati o riscritti: il catalogo non
certifica la conformità dello storico del warehouse.

Il risultato resta `json.Value`: le pipeline e i filtri Record continuano a vedere
traits e properties come JSON, senza uno schema specifico del catalogo. Un campo
nullable assente rimane assente; un null esplicito rimane presente e nullo.
Ordine e duplicati degli array sono conservati, come i campi non dichiarati.

Le conversioni aggiuntive sono ancora rinviate. Per esempio `rating: 3` e una
quantità espressa come String non sono ancora convertiti nei tipi canonici;
la validazione restituisce un errore di dato. Non si interpreta un tipo non
convertibile come dato assente e non si allarga silenziosamente il tipo a JSON.
Le regole future saranno valutate sui casi del catalogo e confrontate con i
mapping senza obbligo di condividere implementazione o tutte le conversioni.

Il collector attualmente scarta l'intero evento che non supera la validazione,
senza una segnalazione dedicata dello scarto. La decisione fra rifiuto e
conservazione del campo non conforme, e la relativa osservabilità, è stata
esplicitamente rinviata da Marco dopo la revisione. Questo incremento non
modifica tale politica. Prima di esporre i dettagli degli errori sarà necessario
anche limitarne i valori e rendere i messaggi adatti all'utente.

Le eventuali tabelle per evento e per gli elementi degli array restano future.
Una rappresentazione a sole colonne potrebbe accorpare assenza e null esplicito:
non è automaticamente equivalente al JSON per gli operatori che li distinguono.

## Verifiche ripetibili

```sh
go test ./core/internal/state -run '^TestWorkspace(EventDefinitions|EventDefinitionSchemas|LookupEventDefinition)$'
go test ./core/internal/collector
go test ./core -run '^(TestEventCatalogSchemaTypes|Test_checkAllowedTypesProfileSchema)$'
```

I test controllano inventario, matching, immutabilità, tipi e semantiche, oltre
al decoder reale con ordini, quantità frazionarie, importi esatti, array aperti,
ricerca strutturata, live null, durate, identify e group. La verifica delle
restrizioni comuni richiama il validatore effettivo del profile schema.
