# Revisione del catalogo eventi Go e della normalizzazione

Revisiona il nuovo incremento sul branch `typed-filter-values`. Il punto di
partenza del catalogo è `28c4d2110`; `1eb56f257` riporta separatamente il commit
`195cf4cb45f9c475da9030a0ced979aefebed258` di main per Duration Float64, adattando
un selettore dei test all'Admin Shoelace di questo branch. Il commit successivo
introduce il catalogo reale descritto sotto. Non modificare codice durante la
revisione; scrivi il risultato in
`filtering/claude/revisione-catalogo-eventi-go.md`.

## Decisioni concordate con Marco

- Il catalogo è compilato in strutture Go, condiviso fra workspace e accessibile
  da `state.Workspace` tramite i due metodi esistenti. Nessun caricamento runtime.
- I tipi delle proprietà sono quelli nativi Krenalis, scelti secondo il significato
  dei campi. Si applicano i criteri del profile schema, consentendo in più gli
  array di oggetti e le proprietà nullable.
- Duration usa Float(64) reale, come stabilito dal commit già presente su main.
- Il passaggio di Measurement a Float64 e l'aggiunta dei secondi sono lavori
  separati: non vanno anticipati. Gli offset video restano Float64 senza quella
  semantica, con unità e riferimento temporale nelle descrizioni.
- `types` non deve essere modificato. La normalizzazione rimane nel collector;
  gli array di oggetti devono preservare campi extra, ordine e duplicati.
- Le conversioni aggiuntive sono ancora rinviate. Non è garantito che coincideranno
  tutte con quelle dei mapping e non condividono codice con essi.
- Le pipeline continuano a vedere traits e properties come `types.JSON()` e valori
  `json.Value`. Lo schema del catalogo non diventa lo schema dei filtri Record.
- Non sono introdotti JSON Schema in ingresso, tabelle per evento, endpoint HTTP,
  tool MCP, configurazione per workspace, ruoli o versionamento delle definizioni.
- Il catalogo non attesta la conformità degli eventi già presenti nel warehouse.

## Fonti e scelte

Leggi `filtering/event-catalog/README.md`, la specifica consegnata e la risposta
alla ricerca sulle proprietà ambigue. I due archivi originali sono conservati:

- `cdp-event-taxonomy.zip`: `cdp-event-catalog.json` per ID, nomi, descrizioni e
  matcher; `cdp-event-bindings.json` per le varianti, che rimangono disabilitate.
- `cdp-event-properties.zip`: `cdp-event-properties.json` autorevole per i campi;
  i 102 JSON Schema dei payload sono derivati. Si usa il profilo `documented`.

Il generatore controlla i manifest, la baseline comune e i 102 ID, riutilizzando
gli helper del pacchetto per espandere i campi condivisi e selezionare le evidenze
del profilo documented. Produce un file Go; non è parte dell'esecuzione dell'app.
Gli altri metadati originali restano negli archivi, senza estendere ora il modello
runtime con tutti i campi della specifica più ampia.

Sono scelte Krenalis esplicite, non dichiarazioni dei provider: Money Decimal(18,4)
senza valuta fissa; quantità Decimal(18,4); conteggi e ordinali Int(32); rating
String; ad_type eterogeneo e query String/Object come JSON; Date per birthday
documentato come data logica. Tipi ignoti o soltanto suggeriti restano JSON.

Le chiavi di esempio `Feed Type` e `total billed` sono fuori dalla grammatica dei
nomi di types.Property. Sono preservate come JSON non dichiarato, non rinominate.
Trigger non viene inventato quando manca una fonte separata. La definizione
storica merge è conservata, ma il decoder continua a rifiutare quel tipo.

## Verifiche richieste

1. Fedeltà delle 102 definizioni e dei matcher; assenza di alias automatici e di
   attivazione involontaria del tipo merge. Generic page/screen con nomi presenti
   o assenti devono funzionare.
2. Scelta dei tipi: cerca casi in cui un criterio troppo generale del generatore
   confonda proprietà omonime con significati diversi, perda null documentati o
   trasformi un esempio in vincolo. Distingui un errore rispetto alle decisioni
   da una preferenza alternativa di prodotto.
3. Correttezza degli schemi condivisi, ownership dei tratti di gruppo rispetto
   alla persona e impossibilità di alterare il catalogo tramite i valori restituiti.
4. Normalizzazione degli oggetti dentro gli array, errori con indice e percorso,
   conservazione di precisione, ordine, duplicati, campi extra e null espliciti.
   Nessuna modifica silenziosa del catalogo dei 26 operatori o di types.
5. Integrazione reale del decoder con Workspace.LookupEventDefinition. I vecchi
   test generici usano una fixture stabile; la nuova suite usa il catalogo reale:
   verifica che questa separazione non nasconda regressioni.
6. Copertura delle scelte più rischiose e riproducibilità della generazione. Non
   presentare rendering o test locali come certificazione Snowflake reale.

Sono già stati eseguiti: suite collector e state, con PostgreSQL reale per state;
test dei tipi del catalogo tramite il validatore effettivo dei profili; regressioni
del profile schema Duration; compilazione core/cmd; go vet; TypeScript; rigenerazione
identica byte per byte; gofmt e git diff --check. Il test MaxMind è saltato perché
manca il database esterno. I test browser Admin non sono stati eseguiti.

Riporta problemi concreti con priorità, riferimenti e controesempi. Non richiedere
di implementare funzionalità espressamente rinviate e non indebolire i test per
accettare una divergenza non intenzionale.
