# Revisione del catalogo eventi Go e della normalizzazione

Revisione di `28c4d2110..e4f28a9e7` sul branch `typed-filter-values`. Non ho
modificato codice. I controesempi sono stati eseguiti con il decoder reale e
`Workspace.LookupEventDefinition`, tramite un file di test caricato con
`go test -overlay` senza toccare il repository.

## Esito

Struttura, matching, ownership dei tratti, immutabilità e normalizzazione degli
array di oggetti sono corretti. Il generatore ha però due errori di tipo rispetto
alle decisioni concordate (P1). Inoltre c'è una conseguenza di prodotto da far
confermare esplicitamente a Marco: oggi un dato non valido scarta l'intero evento
senza lasciare traccia, e con 102 definizioni reali questo riguarda tutto il
traffico identify/page/screen/group/alias di ogni workspace (P1, decisione).

## P1. Errori rispetto alle decisioni

### 1. `identity.identified` `traits.company.id` diventa Float(64)

Entrambi i provider documentano `string` e `number`, e la descrizione dice
"both text and numeric identifiers are documented". In `native_type` il ramo
`kinds <= {"number","integer","string"}` produce String solo se
`kinds == {"string"}` o se il nome è `rating`/`value`. Tutti gli altri campi
misti finiscono in "Altri Number → Float(64)" (`generate_catalog.py:109-134`).

Controesempi verificati:

- `{"type":"identify","userId":"u","traits":{"company":{"id":"acme-42"}}}` →
  `traits.company.id: property "" does not have a valid value: "acme-42"`: l'evento viene scartato.
- `{"company":{"id":12345678901234567}}` è accettato ma riscritto come
  `{"company":{"id":1.2345678901234568e+16}}`: alterazione silenziosa di un
  identificatore memorizzato.

Questo contraddice la scelta dei tipi "secondo il significato" e la conservazione
della precisione. Coerentemente con `query` e `ad_type`, la scelta prudente è
JSON finché la conversione numero → testo resta rinviata, oppure String se Marco
accetta che gli ID numerici vengano rifiutati fino alla conversione.

Il generatore dovrebbe inoltre fallire su ogni combinazione mista di scalari non
elencata esplicitamente, invece di ricadere su Float. Oggi `company.id` è l'unico
caso, ma la regola generica lo ha nascosto.

### 2. `identity.group_associated` `traits.employees` diventa Int(32) contro il tipo documentato

Evidenze: Segment e RudderStack documentano `string` nelle tabelle. `number`
compare solo come `official_example`. L'helper del pacchetto stabilisce che "gli
esempi possono allargare un tipo dichiarato ma non stabilire da soli un tipo
restrittivo", e il README dice "Non si applicano enum o vincoli desunti dagli
esempi". Invece `if name == "employees": return "types.Int(32)"`
(`generate_catalog.py:116-117`) adotta il tipo dell'esempio e rifiuta quello
dichiarato.

Controesempi: `"employees":"42"` e `"employees":"10-50"` → evento group
scartato. Incoerenza collegata: lo stesso concetto in
`identify.traits.company.employee_count` è Float(64), con descrizione "without
an inferred integer constraint".

Il test `TestWorkspaceEventDefinitionSchemas` ("group employee count") fissa la
scelta errata e andrà corretto insieme al generatore, non indebolito. Le voci
`employees` e `logins` in `INTEGER_FIELDS` sono di fatto morte: la prima è
intercettata prima, la seconda ha tipi documentati vuoti e diventa JSON.

## P1 (decisione da confermare). Il rifiuto scarta l'evento in silenzio

`serveEvents` fa `continue` su `ValidationError` (`collector.go:826-833`): niente
conteggio, niente observer, nessuna traccia. Le decisioni concordate rinviano le
conversioni, ma non dicono esplicitamente che un dato non convertibile debba far
perdere l'intero evento. Con il catalogo reale la validazione si applica sempre
a page, screen, identify, group e alias, perché le definizioni generiche
corrispondono a ogni nome, e alle 96 track note.

Payload che prima di questo incremento erano accettati e ora vengono scartati,
tutti verificati:

| Payload | Errore |
|---|---|
| identify `traits.lastName: null` | `value cannot be null` |
| identify `traits.company: "Acme Inc"` | `json: cannot unmarshal JSON string into Go map[string]json.Value` |
| identify `traits.createdAt: 1699999999` | valore non valido |
| identify `traits.birthday: "1990-05-12T00:00:00.000Z"` (`JSON.stringify` di un `Date` JS) | valore non valido |
| identify `traits.phone: 3912345678`, `traits.age: "19"` | valore non valido |
| page `properties.referrer: null` | `value cannot be null` |
| Order Completed `currency: null`, Product Added `price: null` | `value cannot be null` |
| Order Completed `total: 0.30000000000000004` (artefatto float JS) | Decimal(18,4) rifiuta la quinta cifra decimale invece di arrotondare |
| Signed In `value: 10` (`number` documentato da Segment) | String non accetta numeri |
| Checkout Started `value: "10.5"` (`string` documentato da RudderStack) | Money non accetta stringhe |

Distinzione richiesta:

- **Coerenti con le decisioni ma ad alto impatto**: `rating: 3`, Money
  Decimal(18,4), birthday Date, String per `value`. Sono tipi scelti
  esplicitamente, con conversioni rinviate.
- **Incoerenti con le regole del README**: `value` di
  `commerce.checkout_step_completed` e `commerce.payment_info_entered` è
  descritto come "supplied for the GA4 destination mapping". La tabella del README
  manda in JSON i tipi "specifici di una destinazione", invece qui diventa Money,
  scelto da una sottostringa della descrizione ("Monetary value"). L'esempio
  ufficiale RudderStack di quel campo è una stringa, quindi viene rifiutato.

Non chiedo di anticipare le conversioni. Chiedo una decisione esplicita fra
rifiutare l'evento e accettarlo senza normalizzare il campo non conforme.
In ogni caso lo scarto dovrebbe essere osservabile: oggi un workspace che manda
`lastName: null` perde gli identify senza poterlo scoprire.

## P2

### 3. La separazione fixture / catalogo reale nasconde una sola differenza

Ho rieseguito l'intera suite collector sostituendo il fallback della fixture con
il catalogo reale. Fallisce solo `Test_Decoder`, e solo per l'ordine delle chiavi:
`{"name":"bob","age":19}` diventa `{"age":19,"name":"bob"}`, perché le proprietà
dichiarate vengono prima e le altre seguono in ordine alfabetico. Nessuna
regressione di rifiuto è quindi nascosta. Resta non verificato dalla suite
storica che in produzione ogni identify/page/screen/group/alias venga riscritto,
compreso `createdAt: "2026-11-04"` → `"2026-11-04T00:00:00Z"`. Basta documentarlo
in un caso della suite reale.

### 4. Criteri del generatore troppo generali

- I tipi dipendono da sottostringhe delle descrizioni: `"Monetary value"`,
  `"Checkout value"`, `"logical type Date"`. Una riformulazione cambierebbe il
  tipo senza alcun avviso.
- `MONEY_FIELDS`, `INTEGER_FIELDS` e le regole su `email`, `url`, `_url` valgono
  per nome in tutti gli eventi e in tutte le profondità. Oggi non ho trovato
  omonimi mal classificati oltre ai punti 1 e 2: `discount`, `total` e `amount`
  sono tutti monetari, `position` fuori dal video è ordinale. Il criterio però non
  protegge le definizioni future.
- Proposta: una tabella esplicita di eccezioni per `(event_id, path)` e un errore
  per ogni campo con più tipi scalari che non vi compare.

## P3

5. `archive_files` verifica un manifest contenuto nello stesso zip. Rileva la
   corruzione, non la manomissione, eppure il docstring ("before reading any
   executable helpers") lascia intendere una garanzia prima di eseguire l'helper
   con `runpy`. Va solo riformulato.
6. Nessun controllo automatico impedisce la deriva fra generatore e
   `event_catalog_data.go`. Ho rigenerato nello scratchpad con Python 3.13.2 e
   l'output è identico byte per byte.
7. I messaggi di errore contengono `property ""` (nome vuoto, ereditato da
   `types.Decode` su uno scalare), il valore grezzo non limitato e dettagli Go
   (`map[string]json.Value`). Oggi vengono scartati. Se `PropertyPath` verrà
   esposto, vanno ripuliti e limitati.
8. Gli schemi condivisi prendono il nome dal primo evento:
   `videoAdCompletedProperties` serve anche `video.ad_started`, `emailBouncedTraits`
   serve tutti gli eventi email. È solo leggibilità.

## Verificato senza problemi

- **102 definizioni e matcher**: tutte raggiungibili tramite
  `LookupEventDefinition` con ID e metadati della tassonomia. Il match è esatto
  e senza alias. Page e screen generici funzionano con nome presente, assente o
  di un'altra definizione. `merge` resta rifiutato sia dal path (`decoder.go:233-239`)
  sia dal body (`decoder.go:462-467`).
- **Tratti di gruppo**: group usa `identityGroupAssociatedTraits`, senza campi
  della persona. Il `context.traits` di identify e group non viene validato.
  Alias e track validano il `context.traits` spostato in `event["traits"]`.
- **Immutabilità**: `EventDefinitions` clona lo slice e `types.Type` e
  `Properties` restituiscono copie (`Slice`, `Values`, iteratori). Il catalogo
  non è alterabile.
- **Array di oggetti**: ordine, duplicati, campi extra, null espliciti,
  precisione dei Decimal e JSON non dichiarato vengono conservati. I percorsi
  hanno indice, come `properties.products[1].quantity`. Le chiavi duplicate sono
  rifiutate a monte dal parser.
- **Null**: un nullable assente resta assente, un null esplicito resta presente.
- `tools/types`, `core/internal/filters` e i 26 operatori non sono modificati.
- Il backport Duration `1eb56f257` coincide con `195cf4cb` in `core/`. In
  `admin/` cambia solo l'adattamento a Shoelace.
- Precisione e nullable di `total_length`, offset video Float64 senza
  Measurement, `conversation_duration` Float Duration: conformi alle decisioni.

Nulla di questa revisione certifica il comportamento su Snowflake reale:
rendering e test sono locali.
