# Preparazione dell'attivazione Unified su Snowflake

Stato: preparazione locale e revisione mirata concluse; prova live finale ancora
da eseguire.

- Branch: `prepare/unified-snowflake-activation`.
- Worktree: `/private/tmp/krenalis-snowflake-unified-activation`.
- Base: `6b8b726c5bf298250c251f39284bc2bec3f98024`, dopo la chiusura della
  revisione approfondita.
- `typed-filter-values`, nella worktree `/Users/marco/src/cluster`, conserva
  il gate chiuso. Questo incremento non viene trasferito prima della verifica
  live e della revisione mirata.

## Modifiche

Il gate compilato di Snowflake passa a `true`. SQL, letture dei dati, connessioni
di produzione, autenticazione e contratti dei filtri restano quelli revisionati.
Non viene introdotto un interruttore per workspace o un bypass a runtime.
Gli altri cambiamenti al Go di produzione riguardano i tre testi MCP descritti
sotto e il commento del gate, reso uguale nei due provider.

I test delle factory richiedono entrambe le piattaforme verificate. Il rifiuto
di un backend non verificato resta coperto con un doppio esplicito, indipendente
dal valore del gate reale. Il test locale sui guasti del cursore Snowflake ora
esegue il percorso ordinario, senza overlay.

`prepare.py` richiede il gate aperto nel sorgente e non sostituisce
`warehouses/snowflake/profile_filter.go`. Gli overlay mantengono soltanto le
impostazioni e le verifiche di sessione, l'osservazione dei risultati e, quando
richiesto, il server HTTP interno e i tag delle query. I test Python verificano
l'assenza della sostituzione del gate e il rifiuto di un sorgente con gate chiuso.

La guida MCP descrive le capacità abilitate su PostgreSQL e Snowflake.
Lo stesso vale per le istruzioni del server restituite da initialize e per le
descrizioni di query-profiles ed explain-profile-filter in tools/list.
Le evidenze delle campagne precedenti conservano i loro riferimenti e i loro
limiti; non dimostrano l'esecuzione live di questo candidato.

## Verifiche locali del candidato iniziale

Queste evidenze si riferiscono al candidato `72e5cfeaf`, prima delle correzioni
documentate nella sezione successiva.

Le prove usano `KRENALIS_SKIP_SNOWFLAKE_TESTS=true`. Non si caricano credenziali
Snowflake e non si eseguono query remote. Le integrazioni PostgreSQL usano
container locali usa e getta.

I log di questa preparazione sono in
`/private/tmp/krenalis-snowflake-activation-evidence-i0q0hxhi/`.
Gli asset Admin sono una copia isolata di quelli già generati nella campagna
precedente; le prove HTTP non certificano il comportamento dell'interfaccia.

| Verifica | Esito |
| --- | --- |
| Suite brevi con race: snowflake, datastore, core, MCP e snowflaketester | 8.312 PASS, 54 SKIP di integrazione |
| Core PostgreSQL: workspace e servizio pubblico, con race | 66 PASS |
| HTTP PostgreSQL, server interno isolato, con race | 9 PASS |
| MCP PostgreSQL: query e spiegazioni, con race | 6 PASS |
| MCP PostgreSQL: accessi, revoca, catalogo e guida, con race | 230 PASS |
| MCP e krenalistester dopo l'aggiornamento della guida, con race | 37 + 25 PASS |
| Python snowflaketester | 45 test OK |
| Bootstrap pubblico Snowflake con skip esplicito | 2 SKIP attesi, non prove remote |
| Mutante con gate richiuso | Rilevato da TestUnifiedPrepare e TestProfileSchemaFailures |
| Controllo finale delle factory | 2 PASS, 3 SKIP Snowflake attesi |
| Vet, build, gofmt e controllo del diff | Puliti |

I conteggi comprendono test padre e sottotest e non vanno sommati fra suite
sovrapposte. Nei tre blocchi HTTP/MCP accettati i manifest confermano sorgenti
stabili. Gli overlay non sostituiscono il file del gate.
Per `72e5cfeaf`, `final-source-comparison.json` confronta i sorgenti con i tre manifest:
gli hash del codice Go di produzione coincidono; le sole differenze Go sono
commenti di dichiarazione abbreviati nei test. Le factory sono state ricontrollate
dopo questa passata.

Sono conservati anche i tentativi iniziali non accettati: la prima suite Python
ha rilevato quattro asserzioni sul vecchio numero di sostituzioni, poi corrette;
una prova MCP locale era bloccata dai permessi di bind della sandbox; un blocco
HTTP/MCP combinato falliva perché il server interno ammette un solo test
principale per processo. Le ripetizioni corrette passano e hanno log separati.

## Correzioni dopo la revisione del 17 novembre

Il rilievo bloccante è corretto: tre stringhe Go inviate ai client MCP
dichiaravano ancora Snowflake non disponibile. Ora initialize e tools/list
descrivono la disponibilità di filtri e spiegazioni su PostgreSQL e Snowflake.
I test verificano i testi delle risposte HTTP, compresa l'assenza delle vecchie
indicazioni. Tre mutanti ripristinano separatamente i vecchi testi e sono
rilevati dai due test aggiornati.

Le note 1 e 2 sono risolte nelle istruzioni operative del runbook e nel README
Unified, preservando i resoconti storici. Il commento del gate è uguale nei
due provider (nota 3). Il controllo di prepare.py richiede una dichiarazione
ancorata all'inizio della riga; una riga preceduta da `//` non abilita la
preparazione (nota 4). Il piano live descrive un unico script sequenziale, che
si interrompe al primo errore (nota 6).

L'ordine delle guardie dei test remoti resta quello preesistente, già verificato
sicuro nella nota 5 della revisione: le credenziali non vengono lette con skip
attivo. Il commento di TestUnifiedStoredNumbersSnowflake è stato aggiornato;
non viene uniformato il controllo diretto di `-short` fra questi test.

Nuove verifiche, sempre con `KRENALIS_SKIP_SNOWFLAKE_TESTS=true`:

- MCP, risposte initialize e tools/list incluse, con race detector: 37 PASS.
- Adapter PostgreSQL e Snowflake, SQL e guasti del cursore, con race: 9 PASS.
- Python test_prepare: 13 test OK, incluso il gate chiuso con riga commentata.
- Tre mutanti MCP rilevati, uno per ogni vecchio testo.
- Vet dei tre package interessati, gofmt e controllo del diff: puliti.

Log in `/private/tmp/krenalis-snowflake-activation-review-d0opipev/`.
Non sono ripetute le prove pubbliche PostgreSQL o le suite degli oracoli:
la correzione modifica soltanto testi, commenti e il controllo dell'harness.
La prova Snowflake live del candidato corretto resta pendente.

La verifica del 17 novembre approva `cc5496961`: il rilievo bloccante e le
note 1–4 e 6 sono risolti; la scelta sulla nota 5 è confermata come legittima
e sicura. Il revisore ha ripetuto MCP (37 PASS), adapter (9 PASS), factory e
rifiuto dei backend non verificati (2 PASS, 1 SKIP), il mutante delle spiegazioni
e i 13 test Python. Non restano rilievi aperti nella revisione mirata.
Il prossimo passo è la prova live descritta sotto; nessun trasferimento del
candidato avviene prima del suo esito positivo.

## Prova live alla prossima finestra

Eseguire dalla worktree del candidato, solo quando l'accesso live è disponibile
e dopo la revisione. Usare l'ambiente di test dedicato e le credenziali approvate,
senza riportarle in comandi, log o documenti. Preparare una directory nuova e
rigenerare gli overlay dal commit revisionato. I limiti e le politiche del
warehouse di test devono essere verificati dal proprietario dell'ambiente.
Eseguire lo script seguente in Bash.

```sh
set -euo pipefail
cd /private/tmp/krenalis-snowflake-unified-activation
export GOPROXY=off GOSUMDB=off
export GOCACHE=/Users/marco/src/cluster/.gocache
campaign=$(mktemp -d /private/tmp/krenalis-snowflake-activation-live.XXXXXX)
assets=/private/tmp/krenalis-snowflake-activation-evidence-i0q0hxhi/assets

for zone in UTC America/Los_Angeles; do
  slug=${zone//\//-}
  python3 -B test/snowflaketester/prepare.py "$campaign/$slug" \
    --timezone "$zone" --result-format ARROW --http --http-assets "$assets"
  for transport in HTTP MCP; do
    KRENALIS_SKIP_SNOWFLAKE_TESTS=false KRENALIS_TESTS_ADDR=127.0.0.1:2087 \
      python3 -B test/snowflaketester/run.py \
        --require-schemas --require-session-timezone "$zone" \
        --require-session-profile standard --require-result-format ARROW \
        --require-result-tag krenalis-transport/service-query \
        --require-result-tag krenalis-transport/service-explanation \
        --require-result-tag krenalis-transport/service-record \
        "$campaign/$slug-$transport.jsonl" -- \
        -overlay "$campaign/$slug/overlay.json" ./test \
        -run "^TestProfiles${transport}Snowflake$" -race -parallel=1 -p=1 -timeout=10m
  done
done
python3 -B test/snowflaketester/run.py --audit "$campaign"
```

Lo script esegue i blocchi in sequenza e si interrompe al primo errore; prima
di riprenderlo verificare la pulizia e usare nuovi nomi di log. Ogni test
principale viene eseguito in un processo separato. L'overlay deve escludere
il file del gate. I manifest devono
riportare sorgenti stabili, sessioni attese e risultati Arrow per tutti e tre
i tag; ogni blocco deve avere esiti PASS senza SKIP. Verificare separatamente
che non restino schemi creati dalla prova, senza toccare quelli altrui.

Le prove coprono query, spiegazioni e Record tramite HTTP e MCP. Non ripetono
l'intera campagna già conclusa e non ampliano i contratti su numeri, collation,
guasti di rete, formati temporali o limiti delle risorse. I problemi assegnati
a main restano fuori da questo incremento.

Solo dopo esito live positivo e approvazione si trasferisce l'incremento a
`typed-filter-values`. Il merge su main e la preparazione del rilascio con
base e updater restano passi separati.
