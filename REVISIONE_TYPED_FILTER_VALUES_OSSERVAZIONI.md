# Osservazioni sulla revisione di typed-filter-values

Valutazione del 16 novembre 2026 sul commit
`45c6a9f127ead2c0815701454d49e7fece17a168`, confrontato con
json-filter-support (`fece0a984eb1e89c33de71fe16512fcfd4beb262`) e main locale
(`f796b4f57a280bb70767479cb142132c121f7ea9`).

La revisione approfondita di Claude
non segnala difetti nella semantica Unified. Individua sei rilievi da gestire
in questo branch. Alcuni coinvolgono codice preesistente, ma il comportamento
problematico è introdotto o esposto dalle modifiche del branch.

## Rilievi del branch

| Ordine | Rilievo | Valutazione e intervento |
| --- | --- | --- |
| 1 | P1: scarto degli eventi non conformi al catalogo | L'applicazione obbligatoria dei nuovi schemi amplia gli scarti rispetto alla base. Il 16 novembre l'utente ha confermato lo scarto dell'intero evento senza segnalazione osservabile. Il comportamento attuale resta; il primo intervento ne esplicita la compatibilità e aggiunge i casi della revisione ai test. |
| 2 | P2: operandi numerici Record senza limite lessicale | Il secondo intervento limita a 4096 byte i token Decimal e JSON Number, prima della conversione, in `DecodeFilterOperand`. Copre anche array e rilettura dei filtri salvati; i controlli di precisione 76/scala 37 rimangono. I letterali monetari avevano già lo stesso limite. |
| 3 | P2: il test dei writer altera il registro globale | La piattaforma fittizia registrata dal nuovo test fa fallire il test preesistente della risoluzione delle identità. La correzione preferita è isolare il nuovo test evitando la registrazione globale, conservando la copertura dei writer reali. |
| 4 | P3: riordino degli endpoint | Il rumore è introdotto dalla risoluzione del merge. Ripristinare l'ordine della base, mantenendo la nuova rotta della spiegazione. |
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
Gli altri rilievi restano sospesi fino alla revisione di questo intervento.

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
Questo intervento è pronto per la revisione; il test dei writer e le altre
correzioni non sono stati modificati.

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
