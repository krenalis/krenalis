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

## Esito finale

Tutti i rilievi della revisione approfondita sono chiusi. Il revisore ha
approvato i nove commit correttivi da `3a3c0a646` a `d977e0c3c`, ripetendo
le verifiche sugli interventi che modificavano codice e confermando anche
le ultime correzioni documentali.

L'attivazione Snowflake è stata poi revisionata separatamente e verificata live
il 17 novembre: 30 nodi HTTP/MCP passati nei due fusi e inventario indipendente
senza residui. Il gate è abilitato; dettagli nel
[registro dell'attivazione](filtering/SNOWFLAKE_UNIFIED_ACTIVATION.md).
Restano separati il merge su main e la conferma della base di rilascio
`v0.48.0` con la preparazione dell'updater.

## Rilievi del branch

| Ordine | Rilievo | Valutazione e intervento |
| --- | --- | --- |
| 1 | P1: scarto degli eventi non conformi al catalogo | L'applicazione obbligatoria dei nuovi schemi amplia gli scarti rispetto alla base. Il 16 novembre l'utente ha confermato lo scarto dell'intero evento senza segnalazione osservabile. Il comportamento attuale resta; il primo intervento ne esplicita la compatibilità e aggiunge i casi della revisione ai test. |
| 2 | P2: operandi numerici Record senza limite lessicale | Il secondo intervento limita a 4096 byte i token Decimal e JSON Number, prima della conversione, in `DecodeFilterOperand`. Copre anche array e rilettura dei filtri salvati; i controlli di precisione 76/scala 37 rimangono. I letterali monetari avevano già lo stesso limite. |
| 2a | P2: token Decimal nel collector senza limite lessicale | Token scalari Decimal limitati a 4096 byte prima di `types.Decode`, con errore di validazione. I test permanenti proteggono il messaggio e le forme del catalogo. L'ottimizzazione del parser generico resta separata su main. |
| 3 | P2: il test dei writer altera il registro globale | Il test ora usa la piattaforma PostgreSQL esistente e salva le impostazioni cifrate nel workspace di prova. Non registra più una piattaforma fittizia e conserva il percorso reale del datastore e dei writer. Il test preesistente della risoluzione delle identità resta invariato. |
| 4 | P3: riordino degli endpoint | Ripristinato l'ordine della base per la rotta iniziale, le metriche e le organizzazioni. In `cmd/endpoints.go` resta soltanto l'aggiunta della rotta della spiegazione rispetto a json-filter-support. |
| 5 | P3: `Filter.HasCount` senza chiamanti di produzione | Rimosso il metodo e la modalità dedicata a Count in `hasBehavior`. I sette test che lo chiamavano ora verificano soltanto `RequiresUnified`, mantenendo la copertura della visita limitata dell'albero. |
| 6 | P3: documentazione del catalogo inesatta | README e riepilogo sono stati allineati nel P1. Riepilogo e osservazioni conservano la nota di rilascio su scarto e normalizzazione; la voce impropria sul catalogo è rimossa da RELEASE-UPGRADE.md. Completata anche la nota sui limiti delle precedenti credenziali MCP. |

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
  intervento separato. La documentazione chiarisce che i limiti delle
  precedenti credenziali MCP non si trasferiscono automaticamente alla
  connessione principale.

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

Il revisore ha approvato `f9b7dbac7`, confermando l'identità con la base salvo
la nuova rotta e ripetendo i test e i controlli. Il P3 sull'ordine degli
endpoint è chiuso.

## P3: rimozione di Filter.HasCount

Rimosso `Filter.HasCount`, che non aveva chiamanti di produzione. La visita
`hasBehavior` mantiene soltanto la modalità usata da `RequiresUnified`, senza
modificare il riconoscimento delle condizioni o i limiti dell'albero.

Sono aggiornati tutti i sette test chiamanti: sei conservano l'asserzione
su `RequiresUnified`; quello dedicato alla visita è rinominato
`TestFilterRequiresUnified`. Conserva i casi Record, Count, profondità,
ampiezza, nil e cicli, e verifica anche Sum, Average, experience, latestN,
lastVersusPrevious, occurredIn e money. Non restano riferimenti a HasCount
nel codice Go del repository fuori da vendor.

Verifiche:

- `go test -mod=vendor ./core ./cmd -short -race -count=1 -json`:
  2.224 nodi PASS (2.004 core e 220 cmd), 5 SKIP dei test d'integrazione
  esclusi da `-short`, nessun FAIL.
- `go vet -mod=vendor ./core ./cmd`, gofmt dei file modificati e controllo
  del diff: puliti.

Log in `/private/tmp/krenalis-review-p3-hascount-uy858zsm/`.
L'intervento è pronto per la revisione; la documentazione residua resta
rinviata al passo successivo.

Il revisore ha approvato `f95d90b61`, confermando la rimozione completa, la
copertura dei test e le verifiche ripetute. Il P3 su `Filter.HasCount` è chiuso.

## P3: documentazione residua

Il comportamento del catalogo era già stato documentato nel primo intervento.
Il riepilogo e la nota di rilascio in questo documento ne descrivono l'effetto
sull'ingresso in tutti i workspace: scarto dell'intero evento non conforme,
compresa l'elaborazione delle identità, richiesta ancora riuscita e nessuna
segnalazione dedicata. Descrivono anche la normalizzazione dei valori conformi
e precisano che non comporta una migrazione dei dati storici.

README MCP, riepilogo della revisione e note per il rilascio chiariscono che
i limiti del precedente ruolo o warehouse dedicato non si trasferiscono
automaticamente alla connessione principale. Gli amministratori devono
verificare gli statement timeout e il warehouse Snowflake principale con il
suo resource monitor. I budget MCP di argomenti, risposta e rate limit non
limitano la durata o il lavoro eseguito dal warehouse per la singola query.
Il README non afferma più genericamente che i budget del warehouse restano
invariati.

Verifica documentale contro il decoder, il collector e il percorso MCP/core
che usa il contesto della richiesta. Riletti i testi in italiano e inglese;
controllo del diff pulito. Cambiano soltanto quattro file Markdown: nessun
codice Go, SQL, configurazione, base di rilascio o updater. Non sono state
ripetute suite o prove remote per questo intervento documentale.

Questo ultimo intervento è pronto per la revisione. Il gate Snowflake resta
chiuso; attivazione, merge e preparazione del rilascio rimangono separati.

### Correzioni dopo la verifica documentale

Il revisore ha approvato nella sostanza `18055c47a`, chiedendo due precisazioni
sulla voce del catalogo. Rimossa da `RELEASE-UPGRADE.md`: non descrive una
migrazione del database e quel file viene eliminato quando si prepara
l'updater. La nota destinata al rilascio resta nel riepilogo e in questo
documento. La voce MCP resta legata alla rimozione delle colonne.

Gli effetti della normalizzazione sono presentati come esempi, inclusi Float,
Decimal, Time e IP. L'ordine può cambiare per tutte le chiavi degli oggetti:
prima quelle dichiarate nell'ordine dello schema, poi quelle extra in ordine
alfabetico. Verificato contro `normalizeJSONValue` e i casi esistenti di
`TestNormalizeJSON`, compreso `arbitrary extra keys`.

Cambiano soltanto tre file Markdown. Diff pulito; nessuna suite ripetuta per
questa correzione documentale. Il revisore ha approvato `d977e0c3c`, chiudendo
entrambi i rilievi documentali senza altre osservazioni.

## Nota da riportare nel rilascio

Il catalogo predefinito introduce validazione e normalizzazione all'ingresso
in tutti i workspace. Payload prima accettati possono ora essere scartati
integralmente, senza errore nella risposta, log, metriche o osservatore
dedicati. Per esempio un identify con `traits.company: "Acme"` o
`traits.email: null`, una page con `properties.title: null` e un Order Completed
con `properties.total: "10.50"` non rispettano gli schemi. Gli eventi scartati
non contribuiscono neppure all'elaborazione delle identità. Per gli eventi
conformi possono cambiare, per esempio, la rappresentazione di timestamp,
Float, Decimal, Time e IP e l'ordine delle chiavi degli oggetti, comprese
quelle non dichiarate. Non sono previste conversioni aggiuntive né una
rivalidazione degli eventi storici; non serve una migrazione del database.

Il gate Snowflake era chiuso alla conclusione di questa revisione; la successiva
attivazione verificata è registrata nell'esito finale sopra. Il merge su main
resta separato.

## Rinomina in behavioral filtering

Il lavoro prosegue sul branch `behavioral-filters`. Il package interno è
`core/internal/behavioral`; `Filter.HasBehavioralConditions` distingue la presenza
effettiva di condizioni comportamentali. Il primo intervento è nel commit
`acc5fba92`: rinomina codice, test, ancoraggi degli overlay e comandi correnti.
Un confronto strutturale conferma dichiarazioni e corpi equivalenti nei 102
file Go modificati, applicando le sole rinomine dichiarate.

Il codice pubblico `UnifiedFilteringUnavailable` diventa
`BehavioralFilteringUnavailable`, anche nel valore restituito tramite HTTP e
MCP. Endpoint, formato JSON, filtri salvati, SQL e capacità restano invariati.
Il supporto behavioral nelle pipeline di esportazione profili è rinviato.
“Unified profiles” continua a indicare l'unificazione delle identità. Le
evidenze storiche mantengono nomi, commit e hash originali.

Verifiche della rinomina (2026-11-17):

- Suite estesa `-short -race`: 10.058 eventi PASS, 55 SKIP, nessun FAIL.
  Comprende datastore, core, cmd, MCP, il motore behavioral, i due warehouse,
  il compositore SQL condiviso e snowflaketester.
- Integrazioni PostgreSQL aggiuntive con `-race`: 125 PASS e uno SKIP Snowflake;
  writer reali, risoluzione delle identità e servizio profili.
- HTTP/MCP PostgreSQL con overlay rigenerati: rispettivamente 10 e 7 PASS,
  senza FAIL o SKIP. I conteggi JSON includono sottotest e pacchetti.
- Tutti i 45 test Python degli strumenti passano a sorgenti fermi. Un primo
  run aveva rilevato una modifica concorrente dei commenti: il singolo caso
  ripetuto e poi l'intera suite sono passati senza cambiare il test.
- Cinque casi Cuba già noti esclusi dai comandi del package behavioral;
  le loro aspettative restano invariate rispetto alla base.
- Confronto AST finale su 112 file Go: dichiarazioni e corpi equivalenti dopo
  le rinomine e i soli cambi di testo dichiarati. Compresi gli aggiustamenti
  di impaginazione richiesti dai nomi più lunghi.
- `go vet`, compilazione dei pacchetti e dell'eseguibile da `main.go`, gofmt
  e controllo del diff puliti. Il collegamento locale estraneo `connectors.go`
  resta invariato. Nessun nuovo test live Snowflake.
- Sezioni storiche del runbook e del README del motore identiche alla base,
  salvo collegamenti alla nuova posizione del README. Archivio di attivazione,
  initdb, state, Admin e codice di validazione/esecuzione pipeline invariati.

Log e strumenti temporanei: `/private/tmp/krenalis-behavioral-rename-n32x6vu4/`.
