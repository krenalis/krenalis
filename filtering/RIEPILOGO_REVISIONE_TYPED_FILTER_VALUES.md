# typed-filter-values: base per la revisione approfondita

Questo documento prepara la revisione complessiva del branch. Non approva
il merge, il rilascio o l'apertura del gate Snowflake. L'utente ha richiesto
una fase di revisione approfondita prima di valutare il merge.

## Riferimenti

Stato rilevato il 16 novembre 2026, repository `/Users/marco/src/cluster`:

| Riferimento | Commit |
| --- | --- |
| Commit del branch verificato nella campagna live | `4928f87fb70a9ed81a9721406546d44628362d3a` |
| Base del confronto: json-filter-support | `fece0a984eb1e89c33de71fe16512fcfd4beb262` |
| Main locale, prima dell'integrazione attesa | `f796b4f57a280bb70767479cb142132c121f7ea9` |
| Merge json-filter-support nel branch, già revisionato | `8ac0127976b34e9d8e5043eb1300261eb6a57307` |

Quando si valuterà il merge su main, json-filter-support sarà già integrato.
Il confronto funzionale va quindi fatto con quella base, non con il main
locale precedente. Il merge entrante è già concluso; il trasferimento di
typed-filter-values su main resta da decidere dopo la revisione.

Il delta rispetto alla base comprende 402 file al commit indicato. Questo
numero include documentazione, catalogo generato e test: non misura la quantità
di logica da revisionare. Nessun file sotto vendor cambia rispetto alla base;
fra le dipendenze Admin viene aggiunto bignumber.js.

```sh
git -C /Users/marco/src/cluster diff --name-status fece0a984eb1e89c33de71fe16512fcfd4beb262 4928f87fb70a9ed81a9721406546d44628362d3a
```

Year come semantica intera, PropertyRoles e Web Awesome appartengono già alla
base. Le risoluzioni del merge li hanno collegati agli operandi tipizzati e
alle nuove capacità del datastore; non sono funzionalità introdotte da questo
branch. Dopo il merge revisionato sono cambiati soltanto l'ordine di due test
e la documentazione.

## Capacità introdotte

| Area | Comportamento disponibile |
| --- | --- |
| Operandi Record | Il JSON pubblico conserva String, Number, Boolean e Null nei contesti ammessi. Soglie numeriche esatte, senza passaggi intermedi attraverso float64. Boolean e Null JSON sono ammessi nella membership; non sono operandi generici per ogni confronto. |
| Count | Conteggio di messaggi registrati track/page/screen; sorgenti nominate o generiche page/screen, where, minimo, valuta governata e qualificazione money. Nome richiesto nella sorgente: un messaggio senza nome è certamente escluso. |
| Periodi | Durate trascorse, bande temporali, intervalli assoluti semiaperti, periodi completi di calendario e storia registrata. I calendari sono riferimenti espliciti del filtro, non impostazioni del workspace. |
| Sum e Average | Importi governati nella sola valuta nativa richiesta. Sum usa limiti esatti; Average confronta somma e soglia per conteggio senza divisione, con minimo obbligatorio. Le spiegazioni Average mostrano somma e conteggio, non una media arrotondata. |
| First, Last, Nth | Selezione prima della verifica, con candidati possibili, parità, assenza certa/possibile e nessun ripiego su un altro evento. Nth ammette posizioni 1–10. Verify ammette Record, occurredIn e money nei contesti previsti. |
| Latest N | Blocco completo di 1–10 eventi con All, Some o Cardinality. La valutazione conserva i blocchi congiuntamente possibili, non combina candidati indipendenti per posizione. occurredIn non è ammesso nel predicato di questa forma. |
| LastVersusPrevious | Differenza ultimo meno precedente sui due messaggi selezionati nella stessa valuta governata della base. Sono eseguibili i quattro confronti ordinati sulla differenza; gli altri check validi restano non implementati. |
| Spiegazioni | Albero ordinato, esiti MATCH/NO_MATCH/UNKNOWN, periodi risolti, conteggi, limiti monetari e metadati di selezione. Non espongono payload o testimoni degli eventi. Sono nuove valutazioni, non snapshot della query precedente. |
| Catalogo eventi | 102 definizioni predefinite condivise, metadati inglesi e schemi nativi per traits/properties. Lettura tramite core e MCP; il catalogo non certifica la conformità dei dati storici. |
| Ingresso eventi | Lookup della definizione, validazione e normalizzazione dei campi con schema; proprietà non dichiarate conservate. Il valore resta json.Value. Conversioni aggiuntive e validazione JSON Schema non sono introdotte. |
| Discovery | Elenco delle sorgenti osservate e ispezione limitata di percorsi/tipi/presenza. Nessun valore di esempio, payload o identificativo; nessun attraversamento degli array. |

Gli importi si riferiscono ai messaggi Order Completed registrati. Non ci sono
conversioni valutarie o deduplica degli ordini commerciali. Una restrizione a
EUR esclude le altre valute: non rappresenta automaticamente la spesa totale.

Count, Sum e Average compongono minimo e test principale separatamente con
AND Kleene. Questa scelta è confermata; la valutazione congiunta rinviata non
va reintrodotta durante la revisione.

I limiti della grammatica e delle dipendenze sono parte del contratto: una
richiesta valida ma non eseguibile produce un errore di capacità. Una richiesta
invalida prevale su una capacità mancante, indipendentemente dall'ordine dei
rami. UNKNOWN è un esito dei dati, non un ripiego per errori tecnici.

Le where di Sum/Average e della base delle esperienze restano solo Record.
occurredIn è ammesso soltanto nel verify della singola esperienza. Money è
ammesso nei contesti descritti dalla guida, con i vincoli sul numero di foglie
e sulle letture raw sovrapposte; non è un predicato generico del filtro radice.

## Superficie HTTP, MCP e Admin

- `GET /v1/profiles` usa `Workspace.QueryProfiles` per la forma pubblica dei
  filtri. Le richieste solo Record conservano il percorso dedicato.
- `GET /v1/profiles/{kpid}/filter-explanation` espone la spiegazione.
- MCP offre `query-profiles`, `explain-profile-filter`, `profile-filter-doc`,
  `list-event-definitions`, `get-event-definition`,
  `list-observed-event-sources` e `inspect-event-fields`, oltre agli strumenti
  di schema già presenti. Non esegue SQL fornito dal client.
- L'autenticazione resta per chiave MCP del workspace, con rate limit. Le
  richieste usano la connessione principale. OAuth e un nuovo interruttore di
  abilitazione del workspace non fanno parte del branch.
- Le vecchie credenziali MCP e i package readonly sono rimossi. Le chiavi
  esistenti restano valide; i GRANT delle precedenti credenziali readonly non
  limitano più le proprietà interrogabili. Questo cambio di accesso deve
  essere comunicato agli amministratori prima dell'aggiornamento.
- L'Admin conserva l'editor Record con il nuovo trasporto degli operandi.
  Non viene aggiunta una UI per le nuove primitive Unified.

## Compatibilità e rilascio

Il formato di una richiesta API e quello di `pipelines.filter` sono distinti.
I vecchi operandi JSON persistiti come stringhe vengono letti come JSON String;
non si inferisce il tipo Number né si riscrive l'albero per ricostruire il
precedente doppio significato. L'utente ha accettato possibili differenze nei
risultati, anche per valori assenti o non applicabili. Non è richiesta una
compatibilità esatta con i filtri precedenti.

Il commit `1e95ed5bf` e RELEASE-UPGRADE.md registrano questa decisione. Lo studio
di una migrazione conservativa è stato interrotto: non è un'attività pendente.

Il catalogo introduce anche una modifica di compatibilità nell'ingresso eventi,
comune a tutti i workspace. Il decoder valida e normalizza traits e properties
quando la definizione corrispondente ne specifica lo schema. Il 16 novembre
l'utente ha confermato lo scarto dell'intero evento non conforme, senza log,
metriche, osservatore o errore al client dedicati: la richiesta resta riuscita.
Si perde anche l'elaborazione delle identità di quell'evento. I payload conformi
possono cambiare rappresentazione, per esempio date in UTC, decimali canonici
e ordine delle chiavi. Questo effetto va comunicato nelle note di rilascio;
gli eventi storici non vengono rivalidati o riscritti.

Rispetto a json-filter-support, la modifica strutturale di schema.sql è la
rimozione delle due colonne delle credenziali MCP. Le note di rilascio mantengono
anche i passi Year già previsti dalla base. La base dichiarata nelle note è
v0.48.0; la sua conferma e l'updater appartengono alla preparazione del rilascio.
Non è stato scritto un updater. L'aggiornamento avverrà con tutti i nodi fermi
e poi avviati sulla stessa versione, secondo la decisione dell'utente.

I dati dei fusi seguono il comportamento standard di Go: sistema e fallback
time/tzdata. Nessun archivio incluso o nuova revisione è richiesto; gli offset
della revisione 1 seguono il database del runtime. I test Cuba noti non vanno
corretti o resi tolleranti come parte di questa preparazione.

## Aree da sottoporre alla revisione approfondita

La revisione del merge entrante non sostituisce la revisione di tutto questo
delta. Le revisioni incrementali sono evidenze utili, non una prova globale.

| Area | Verifiche richieste |
| --- | --- |
| Codec e validazione | Grammatica chiusa, duplicati/null, conteggi lessicali, precisione 76/scala 37, arità, percorsi e limiti. Chiamate dirette a core/unified/warehouses devono rispettare le stesse garanzie del JSON pubblico. |
| Semantica | Ternarietà, assenza/null/tipo inapplicabile, composizione AND/OR, minimo Kleene, intervalli semiaperti, stesso T e calendari. Confrontare gli esiti con le specifiche del sottoinsieme ammesso. |
| Selezioni | Parità senza spareggi tecnici, scenari vuoti, N=1 e N=10, blocchi compatibili, coppie ordinate e importi illeggibili. Gli oracoli devono enumerare gli scenari senza replicare la formula SQL. |
| Dipendenze governate | Valuta e importo dalla stessa definizione, condizionamento base/verify/where, sovrapposizioni raw, money ripetute e gruppi annidati. Nessun UNKNOWN evitabile ottenuto perdendo una dipendenza dichiarata. |
| Confine warehouse | Limiti prima di allocare/decodificare, overflow, righe/metadati incongrui, payload grandi, ordinamento e totali, guasti delle copie. I controlli proteggono Krenalis secondo AGENTS.md, senza inventare un controllo generale d'integrità dei dati del workspace. |
| Errori e risorse | Nessun risultato parziale; errori tecnici distinti da UNKNOWN, propagazione opaca negli endpoint, cancellazione/timeout, chiusura dei cursori e ripresa del pool. |
| Wrapper datastore | Conservazione delle sole capacità opzionali reali, gate inoltrato, errori avvolti una sola volta e factory effettiva attraversata dai test pubblici. |
| MCP | Scoping della chiave, revoca, rate limit, connessione principale, budget HTTP/argomenti/risposta, numeri esatti nel filtro testuale, classificazioni degli errori e assenza di fallback SQL. |
| Catalogo e collector | Matcher e schemi coerenti, metadati non mutabili dall'esterno, array/object/nullable, campi aggiuntivi e json.Value preservati. Verificare anche l'effetto del catalogo sull'accettazione degli eventi. |
| Discovery | Intervallo e campione limitati, confronti letterali, nome nullo/vuoto, contatori di presenza e dati non ispezionati, troncamento esplicito e assenza di valori restituiti. |
| Admin | Precisione nel browser e nel trasporto, richieste e query string, editor/parser, Sync-Token e mancata reintroduzione delle impostazioni MCP ritirate. |
| Documentazione | Guida MCP e messaggi devono descrivere le capacità effettive, i significati di ordine/spesa, i limiti e i cambiamenti di accesso. Distinguere le evidenze storiche dalla verifica del codice integrato. |

Nel collector, gli errori di validazione fanno proseguire il ciclo senza
inoltrare l'evento. Lo scarto senza segnalazione osservabile è ora una scelta
esplicita dell'utente, confermata il 16 novembre 2026 dopo la revisione
approfondita. Le conseguenze sono descritte nella compatibilità di rilascio.

## Verifiche già disponibili

I conteggi Go includono test padre e sottotest; non sono conteggi di asserzioni
e non vanno sommati fra esecuzioni sovrapposte.

| Evidenza | Risultato e limite |
| --- | --- |
| Suite breve selezionata sul merge | 17.292 PASS, 139 SKIP; escluse le prove Cuba note, tre connettori con credenziali esterne e il caso ClickHouse intermittente. Non è una suite completa senza esclusioni. |
| PostgreSQL con race detector | Ripetizione finale: 1.260 PASS, 1 SKIP. Sette oracoli passati nella prima esecuzione, sei ripetuti nella finale; Latest N è documentato separatamente. |
| HTTP/MCP con race detector | Sei gruppi finali, 287 nodi PASS. Overlay limitato all'avvio del tester e agli asset locali. |
| Admin | Suite Playwright completa con auth: 218 PASS. Due selettori CSS successivi hanno avuto rigenerazione e verifica degli asset; la suite completa non è stata ripetuta per quelli. |
| Runner Snowflake locale | 44 test PASS; nessun collegamento remoto. |
| Qualità e generazione | Build, vet, gofmt, TypeScript, Prettier e controllo generatore del catalogo passati nelle esecuzioni documentate. |
| Revisione indipendente del merge | 481 nodi mirati con race, 1.764 numerici, 102 schema/normalizzatore e TypeScript passati; P3 sull'ordine dei test corretto in `2858693db`. |
| Preparazione post-merge | Overlay Arrow normale e con guasto preparati e compilati, anche con tag e race. Compilazione soltanto: nessuna prova remota o test di comportamento. |
| Snowflake live post-merge, 16 novembre | 17 blocchi riusciti con race detector, più inventario indipendente. Conformance completa UTC: 435 nodi distinti; sottoinsieme LA: 22. HTTP e MCP in entrambi i fusi; chunk, enum, Record, identità, risorse, discovery e interruzioni passati. Dettagli e limiti nel runbook. |

Il rapporto dettagliato del merge e quello indipendente sono rispettivamente:

- `/Users/marco/src/cluster/filtering/claude/risoluzione-merge-json-filter-support.md`;
- `/Users/marco/src/cluster/filtering/claude/revisione-merge-json-filter-support.md`.

I log locali sono sotto `/private/tmp/krenalis-json-merge-study.Vl2MfT/` e
`/private/tmp/krenalis-json-merge-review/`. La preparazione post-merge è in
`/private/tmp/krenalis-unified-postmerge-local-zafpud2t/preflight.json`.
Sono artefatti locali temporanei, non sostituti dei test ripetibili nel repository.

Le esecuzioni live precedenti, con commit e formati osservati, sono riportate
nel runbook. Non vanno presentate come nuove prove remote dopo il merge.
La campagna del 16 novembre verifica invece il codice integrato. Il commit
`45c6a9f12` aggiorna solo questo documento, il README Unified e il runbook.
Il successivo intervento sul P1 aggiunge test e documenta la politica di scarto
confermata dall'utente, senza cambiare il comportamento di produzione.

## Snowflake e lavori separati

La campagna live prevista è conclusa. Il gate `profileFilterVerified` resta
`false`; revisione approfondita, attivazione e merge sono passi separati.
Le tre prove prioritarie sono passate:

1. `TestUnifiedResourcesSnowflake/chunk_payload`: almeno due chunk remoti e
   verifica di tutti i payload al limite esatto.
2. `TestUnifiedChunkFaultSnowflake`: retry esauriti dopo consegna di righe,
   nessun risultato parziale e ripresa sullo stesso pool.
3. `TestUnifiedRecordEnumSnowflake`: confronti contains/does not contain sulle
   colonne degli eventi e sulle colonne String con valori ammessi.

È passata anche la regressione remota selezionata del codice integrato,
compresi i percorsi pubblici HTTP/MCP. Tutti i 29 schemi creati sono stati
eliminati; l'inventario indipendente conferma zero residui della campagna.
Il runbook conserva il primo tentativo interrotto dal timeout di 15 minuti,
escluso dai blocchi riusciti, e le evidenze della sua pulizia.

Log, manifest e overlay sono conservati anche nell'archivio privato
`/Users/marco/src/krenalis-test-evidence/snowflake-completion-20261116-4928f87fb.tar.gz`.
Hash e risultati sono nel runbook. Restano espliciti i limiti di copertura su
guasti di rete, chunk nelle spiegazioni/HTTP/MCP, formati temporali e controlli
ambientali delle risorse. Le prove riuscite non approvano l'attivazione.

Sono esclusi da codice e nuovi studi su questo branch:

- collation degli identificativi tecnici nelle operazioni gestite Snowflake,
  difetto preesistente seguito dall'utente su main;
- trasporto JSON delle colonne native FLOAT/BOOLEAN, seguito su main nella
  [issue #2770](https://github.com/krenalis/krenalis/issues/2770).

Non sono decisioni pendenti o prerequisiti di attivazione di questo branch.
Le correzioni letterali già presenti nel lettore Unified restano. Le prove
JSON native fallite restano tali; l'esclusione non certifica equivalenza dei
trasporti. Anche BeforeClose resta una correzione indipendente da ricevere
quando sarà integrata su main.

## Documenti di riferimento e risultato della revisione

Le specifiche sono:

- `/Users/marco/src/cluster/filtering/06-specifica-semantica-riconciliata-v1.2.md`;
- `/Users/marco/src/cluster/filtering/07-ast-canonico-riconciliato-v1.2.md`;
- `/Users/marco/src/cluster/filtering/08-ast-canonico-modello-finale-estratto.md`.

Per il sottoinsieme eseguibile e le decisioni successive usare anche:

- `/Users/marco/src/cluster/core/internal/unified/README.md`;
- `/Users/marco/src/cluster/cmd/internal/mcp/profile_filter.md`;
- `/Users/marco/src/cluster/cmd/internal/mcp/README.md`;
- `/Users/marco/src/cluster/test/snowflaketester/UNIFIED.md`;
- `/Users/marco/src/cluster/core/internal/initdb/RELEASE-UPGRADE.md`.

La revisione deve produrre rilievi riproducibili con file/righe, effetto sul
contratto, prova o controesempio e correzione minima. Deve distinguere difetti
del delta, problemi della base e limiti esplicitamente accettati. Le prove
indipendenti vanno separate dal semplice esame dei log esistenti.

La valutazione finale deve tenere separati: correttezza del branch, questioni
di prodotto ancora aperte, preparazione del rilascio e verifica Snowflake.
Nessun merge, push, PR o apertura del gate è autorizzato da questo riepilogo.
