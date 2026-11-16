# Osservazioni sulla revisione di typed-filter-values

Valutazione del 16 novembre 2026 sul commit
`45c6a9f127ead2c0815701454d49e7fece17a168`, confrontato con
json-filter-support (`fece0a984eb1e89c33de71fe16512fcfd4beb262`) e main locale
(`f796b4f57a280bb70767479cb142132c121f7ea9`).

La revisione approfondita di Claude
non segnala difetti nella semantica Unified. Individua sei rilievi da gestire
in questo branch. Alcuni coinvolgono codice preesistente, ma il comportamento
problematico è introdotto o esposto dalle modifiche del branch.
Nella verifica del secondo intervento il revisore ha aggiunto un P2 analogo
sulla conversione dei Decimal del catalogo nel collector.

## Rilievi del branch

| Ordine | Rilievo | Valutazione e intervento |
| --- | --- | --- |
| 1 | P1: scarto degli eventi non conformi al catalogo | L'applicazione obbligatoria dei nuovi schemi amplia gli scarti rispetto alla base. Il 16 novembre l'utente ha confermato lo scarto dell'intero evento senza segnalazione osservabile. Il comportamento attuale resta; il primo intervento ne esplicita la compatibilità e aggiunge i casi della revisione ai test. |
| 2 | P2: operandi numerici Record senza limite lessicale | Il secondo intervento limita a 4096 byte i token Decimal e JSON Number, prima della conversione, in `DecodeFilterOperand`. Copre anche array e rilettura dei filtri salvati; i controlli di precisione 76/scala 37 rimangono. I letterali monetari avevano già lo stesso limite. |
| 2a | P2: token Decimal nel collector senza limite lessicale | Anche questo percorso nasce nel branch: la base manteneva properties come JSON grezzo. Limitare a 4096 byte i token scalari Decimal prima di `types.Decode`, restituendo un errore di validazione. L'ottimizzazione del parser generico resta separata su main. |
| 3 | P2: il test dei writer altera il registro globale | Il test ora usa la piattaforma PostgreSQL esistente e salva le impostazioni cifrate nel workspace di prova. Non registra più una piattaforma fittizia e conserva il percorso reale del datastore e dei writer. Il test preesistente della risoluzione delle identità resta invariato. |
| 4 | P3: riordino degli endpoint | Ripristinato l'ordine della base per la rotta iniziale, le metriche e le organizzazioni. In `cmd/endpoints.go` resta soltanto l'aggiunta della rotta della spiegazione rispetto a json-filter-support. |
| 5 | P3: `Filter.HasCount` senza chiamanti di produzione | Il metodo nasce nel branch ed è stato sostituito da `RequiresUnified`. Rimuovere l'API inutilizzata e semplificare la visita; aggiornare tutti i test che lo chiamano. |
| 6 | P3: documentazione del catalogo inesatta | Il README MCP nega effetti sull'ingresso, ma il decoder valida e normalizza. Documentare il comportamento effettivo e le conseguenze di compatibilità secondo la decisione sul P1. |

## Interventi generali separati

- **Costo di `decimal.Parse`: main.** Il file `tools/decimal/decimal.go` è
  identico nei tre riferimenti confrontati. Il costo del parser è preesistente;
  il limite degli operandi Record resta comunque una correzione del branch.
  Il problema generale è registrato nella
  [issue #2774](https://github.com/krenalis/krenalis/issues/2774).
- **Osservabilità generale degli scarti del collector:** possibile lavoro
  separato su main. Gli envelope malformati erano già saltati senza una
  segnalazione dedicata; questo non risolve la nuova politica del catalogo.
- **Timeout e governo delle risorse:** una politica generale richiede un
  intervento separato. In questo branch va documentato che i limiti delle
  precedenti credenziali MCP non si trasferiscono alla connessione principale.

Nessuno dei sei rilievi richiede una correzione su json-filter-support.
Restano fuori dal branch la collation degli identificativi gestiti Snowflake
e il trasporto JSON delle colonne native FLOAT/BOOLEAN, già demandati a main.
Le decisioni su minimo Kleene, vecchi operandi JSON String e dati dei fusi
rimangono quelle già confermate.

## Ordine di lavoro

L'utente richiede un intervento alla volta e una pausa per la revisione prima
di passare al successivo. Si parte dal P1, con questa scelta esplicita:

> Scartiamo l'evento senza segnalazione osservabile.

Il primo intervento conserva il comportamento del decoder e del collector.
Documenta che la richiesta HTTP rimane riuscita anche quando scarta un evento,
che quell'evento non raggiunge pipeline e identità e che i batch proseguono
con gli altri eventi. I test coprono i 14 esempi della revisione con il
catalogo di produzione, verificando l'errore e l'assenza dell'evento decodificato.
La documentazione del catalogo segnalata nel P3 viene allineata per la parte
necessaria al P1. Il revisore ha approvato il primo intervento, commit
`3a3c0a646`, senza rilievi bloccanti. La posizione di questo documento nella
root è richiesta dall'utente; il link al rapporto locale non tracciato è rimosso.

## Verifiche del primo intervento

- `go test -mod=vendor ./core/internal/collector -short -race -count=1 -json`:
  313 nodi PASS, compresi 47 casi del catalogo di produzione. Un solo SKIP:
  l'integrazione MaxMind richiede `KRENALIS_TEST_MAXMIND_DB_PATH`.
- `go vet -mod=vendor ./core/internal/collector`: passato.
- `gofmt` sul test modificato e `git diff --check`: puliti.

Sono aggiunti 14 casi di rifiuto; nessun file Go di produzione cambia.
I log locali sono in `/private/tmp/krenalis-review-p1-agkcn7ee/collector.jsonl`.
Il revisore ha ripetuto test e controlli in una worktree pulita, confermando
questi risultati. Il primo intervento è approvato.

## Secondo intervento: limite degli operandi numerici

`DecodeFilterOperand` controlla la lunghezza dei token Decimal e JSON Number
prima di chiamare il parser decimale. Il limite è 4096 byte, dopo la rimozione
degli spazi esterni. I token entro il limite restano soggetti ai normali
controlli di precisione e scala. La conversione e la gestione degli altri tipi
non cambiano; il parser generico resta oggetto della issue #2774 su main.

I test coprono il confine 4096/4097, zeri rappresentabili con molti decimali,
numeri che superano la precisione, spazi esterni, array e filtri persistiti.
Una String JSON lunga non viene trattata come un Number. Sonde e mutanti
esterni verificano anche l'ordine del controllo rispetto alla conversione.
Il revisore ha approvato il commit `bad4e815f` per questo percorso, confermando
anche le prove e segnalando il residuo nel collector descritto sotto.

### Verifiche del secondo intervento

- Test mirati con race detector: 89 PASS.
- `go test -mod=vendor ./core/internal/state ./core ./cmd/internal/mcp -short -race -count=1 -json`:
  2.595 PASS (561 state, 1.997 core, 37 MCP) e 5 SKIP dei test d'integrazione
  esclusi da `-short`. I test PostgreSQL di state usano container locali.
- Una sonda esterna interrompe il test se un token oltre 4096 byte raggiunge
  `json.Value.Decimal`: 25 PASS con l'implementazione corretta.
- Tre mutanti sono rilevati: eliminazione delle guardie, uso di `>= 4096`
  e controllo spostato dopo il parsing. Quest'ultimo è eseguito con la sonda
  precedente, senza misurare tempi di esecuzione.
- Vet dei tre pacchetti, gofmt dei file Go modificati e controllo del diff:
  puliti.

La prima regressione era stata bloccata dai permessi della sandbox su Docker
e sulle porte locali; la ripetizione autorizzata è quella riuscita. Log,
overlay e risultati dei mutanti sono in
`/private/tmp/krenalis-review-p2-numbers-meq8snaj/`.
Questo intervento è approvato; il test dei writer e le altre correzioni
non sono stati modificati.

## Residuo P2: Decimal del catalogo nel collector

Il limite va applicato anche qui: la nuova normalizzazione del catalogo
espone il costo del parser a payload che nella base restavano JSON grezzo.
La correzione è nel branch, distinta dalla issue generale #2774 su main.

`normalizeJSONValue` verifica i 4096 byte del token scalare Decimal, dopo la
rimozione degli spazi esterni e prima di `types.Decode`. Il superamento produce
un `jsonValidationError`, con il percorso completo del campo. L'evento viene
scartato secondo la politica già confermata. Il controllo raggiunge anche i
campi Decimal degli oggetti in array, come `products[0].price`. Non modifica
il package types o il parser generico, né limita JSON opaco e campi extra.

I test coprono il confine 4096/4097 e gli spazi esterni nel normalizzatore,
gli stessi limiti sul totale con il catalogo di produzione, un prezzo annidato
e un totale di 490.000 cifre entro il budget della richiesta. Controllano anche
che JSON opaco e campi extra mantengano numeri oltre il limite dei Decimal.
Il test dei writer e gli altri rilievi restano sospesi fino alla revisione
di questo intervento.

### Verifiche del residuo nel collector

- `go test -mod=vendor ./core/internal/collector -short -race -count=1 -json`:
  322 PASS, compresi 51 casi del catalogo di produzione; 1 SKIP MaxMind per
  `KRENALIS_TEST_MAXMIND_DB_PATH` assente.
- Una sonda esterna interrompe l'esecuzione se `types.Decode` passa al parser
  decimale un token oltre 4096 byte: 132 PASS sui test del normalizzatore e
  del catalogo, incluso il totale di 490.000 cifre.
- La stessa sonda rileva guardia rimossa e controllo spostato dopo il parsing;
  il mutante `>= 4096` fallisce sui valori validi al limite.
- Vet del collector, gofmt e controllo del diff: puliti.

Log, overlay e risultati sono in
`/private/tmp/krenalis-review-p2-collector-bx6aklni/`.
Il residuo P2 è pronto per la revisione; gli interventi successivi non sono
iniziati.

### Approvazione e verifiche permanenti del residuo

Il revisore ha approvato `fef459bec`, ripetendo le verifiche e confermando
che il controllo copre tutte le forme attuali del catalogo. Le due note
non bloccanti vengono affrontate con soli test, prima di proseguire:

- Un token di 4097 cifre `7` deve produrre l'errore di lunghezza, non quello
  del decoder sul valore fuori range. Questo rileva il controllo spostato
  dopo la gestione dell'errore di `types.Decode`; non dimostra da solo che
  la conversione non sia stata eseguita. La sonda esterna resta la verifica
  dell'ingresso nel parser.
- Un test del catalogo attraversa traits, properties e i contenitori. Un
  Decimal è ammesso solo sui percorsi che raggiungono il normalizzatore e
  la sua guardia, compresi gli oggetti negli array. Se in futuro comparissero
  Decimal dentro map o array delegati interamente a `types.Decode`, il test
  richiederebbe di estendere la protezione. Gli altri tipi restano ammessi.

Nessun file Go di produzione cambia in questo completamento. Il test dei
writer e le altre correzioni restano sospesi per la revisione.

Verifiche del completamento:

- Suite del collector con `-short -race -count=1`: 324 PASS e 1 SKIP MaxMind.
- Il controllo del catalogo visita 152 posizioni Decimal negli schemi runtime
  di traits e properties, tutte protette dalla normalizzazione scalare.
- Un mutante che sposta la guardia dopo il ritorno dell'errore di decodifica
  fallisce sul nuovo controllo del messaggio.
- Quattro sonde con cataloghi alterati sono rifiutate dal test delle forme:
  array di Decimal, map di Decimal, array annidati con oggetti Decimal e
  map di oggetti con campi Decimal. Non sono errori di compilazione.
- Vet, gofmt e controllo del diff: puliti. Evidenze locali in
  `/private/tmp/krenalis-review-p2-collector-tests-4s91ov0j/`.

Il revisore ha approvato `50bbfb7bf`, ripetendo la suite e i mutanti. Il
residuo P2 del collector è chiuso; si passa all'isolamento del test dei writer.

## P2: isolamento del test dei writer

`TestUnifiedWritersPostgreSQL` non registra più la piattaforma fittizia nel
registro globale. Il workspace di prova usa PostgreSQL e impostazioni cifrate
con il normale Cipher e il KMS locale del test. Il datastore crea il provider
tramite la factory di produzione e carica le impostazioni dallo state;
writer, pubblicazione delle identità e query Unified restano reali.

La correzione riguarda solo il nuovo test. Non modifica il registro, il
datastore di produzione né il test preesistente della risoluzione delle
identità. I rilievi successivi restano sospesi fino alla revisione.

Verifiche:

- Prima della correzione, l'esecuzione congiunta dei test dei writer e della
  risoluzione delle identità riproduce il panic sulla piattaforma fittizia.
- Dopo la correzione, la stessa esecuzione con `-race -count=1` passa:
  57 nodi PASS e 1 SKIP, quello Snowflake disabilitato esplicitamente.
- `go test -mod=vendor ./core/internal/datastore -race -count=1 -timeout=20m -json`,
  senza `-short` e senza selettori: 6.171 nodi PASS, 16 SKIP Snowflake, nessun
  FAIL. Il pacchetto completa l'esecuzione in circa 386 secondi.
- `go vet -mod=vendor ./core/internal/datastore`, gofmt e controllo del diff:
  puliti.

Entrambe le esecuzioni successive alla correzione usano
`KRENALIS_SKIP_SNOWFLAKE_TESTS=true` e PostgreSQL locale usa e getta. Non sono
state lette credenziali Snowflake né eseguite prove remote. Log del
controesempio e delle verifiche in
`/private/tmp/krenalis-review-p2-writers-28hq4gqd/`.
L'intervento è pronto per la revisione; gli altri rilievi non sono modificati.

Il revisore ha approvato `7a8ec77e3`, confermando la copertura dei writer e
ripetendo la suite completa con gli stessi risultati. Il P2 sull'isolamento
del test dei writer è chiuso.

## P3: ordine degli endpoint

Ripristinato in `cmd/endpoints.go` l'ordine precedente al merge: `GET /{$}`
prima delle altre rotte GET, le metriche della risoluzione delle identità
prima di quelle delle identità e `/organizations/{id}` prima di
`/organizations`. La rotta `filter-explanation` resta nella posizione attuale.

Verifiche:

- I 123 abbinamenti tra rotte e handler sono identici a quelli del parent.
  Rimuovendo la sola riga `filter-explanation`, il file è identico byte per
  byte alla base json-filter-support `fece0a984`.
- `go test -mod=vendor ./cmd -short -race -count=1 -json`: 220 nodi PASS,
  nessun FAIL o SKIP.
- `go vet -mod=vendor ./cmd`, gofmt e controllo del diff: puliti.

Log e risultato del confronto in
`/private/tmp/krenalis-review-p3-endpoints-he953412/`.
L'intervento è pronto per la revisione. `Filter.HasCount` e la documentazione
residua non sono stati modificati.

## Nota da riportare nel rilascio

Il catalogo predefinito introduce validazione e normalizzazione all'ingresso
in tutti i workspace. Payload prima accettati possono ora essere scartati
integralmente, senza errore nella risposta, log, metriche o osservatore
dedicati. Per esempio un identify con `traits.company: "Acme"` o
`traits.email: null`, una page con `properties.title: null` e un Order Completed
con `properties.total: "10.50"` non rispettano gli schemi. Gli eventi scartati
non contribuiscono neppure all'elaborazione delle identità. Per gli eventi
conformi possono cambiare la rappresentazione delle date, dei decimali e
l'ordine delle chiavi. Non sono previste conversioni aggiuntive né una
rivalidazione degli eventi storici; non serve una migrazione del database.

Il gate Snowflake resta chiuso. Revisione, attivazione e merge sono passi
separati.
