# Osservazioni sulla revisione di typed-filter-values

Valutazione del 16 novembre 2026 sul commit
`45c6a9f127ead2c0815701454d49e7fece17a168`, confrontato con
json-filter-support (`fece0a984eb1e89c33de71fe16512fcfd4beb262`) e main locale
(`f796b4f57a280bb70767479cb142132c121f7ea9`).

La [revisione approfondita di Claude](filtering/claude/revisione-approfondita-typed-filter-values.md)
non segnala difetti nella semantica Unified. Individua sei rilievi da gestire
in questo branch. Alcuni coinvolgono codice preesistente, ma il comportamento
problematico è introdotto o esposto dalle modifiche del branch.

## Rilievi del branch

| Ordine | Rilievo | Valutazione e intervento |
| --- | --- | --- |
| 1 | P1: scarto degli eventi non conformi al catalogo | L'applicazione obbligatoria dei nuovi schemi amplia gli scarti rispetto alla base. Il 16 novembre l'utente ha confermato lo scarto dell'intero evento senza segnalazione osservabile. Il comportamento attuale resta; il primo intervento ne esplicita la compatibilità e aggiunge i casi della revisione ai test. |
| 2 | P2: operandi numerici Record senza limite lessicale | Main e la base limitavano anche gli operandi numerici a 80 caratteri. Il nuovo percorso tipizzato ha perso questa protezione. Il limite va applicato prima della conversione decimale; i letterali monetari hanno già un controllo di 4096 byte. |
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
necessaria al P1. Gli altri interventi restano da affrontare dopo la revisione
di questo primo passo.

## Verifiche del primo intervento

- `go test -mod=vendor ./core/internal/collector -short -race -count=1 -json`:
  313 nodi PASS, compresi 47 casi del catalogo di produzione. Un solo SKIP:
  l'integrazione MaxMind richiede `KRENALIS_TEST_MAXMIND_DB_PATH`.
- `go vet -mod=vendor ./core/internal/collector`: passato.
- `gofmt` sul test modificato e `git diff --check`: puliti.

Sono aggiunti 14 casi di rifiuto; nessun file Go di produzione cambia.
I log locali sono in `/private/tmp/krenalis-review-p1-agkcn7ee/collector.jsonl`.
Il primo intervento è pronto per la revisione. Non sono iniziati gli altri.

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
