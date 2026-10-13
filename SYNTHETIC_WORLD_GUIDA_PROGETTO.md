# Krenalis — Synthetic World: guida del progetto e della realizzazione

**Punto di ingresso per una nuova sessione con accesso al repository.**

Stato verificato all’**11 ottobre 2026** sul branch `synthetic-world`.
HEAD: `50ed99476cf22a8137d1adcf8e2c82f7235e0ec6` — `Integrate six socioeconomic fields into simulated systems`.
Prima di questo aggiornamento documentale, worktree tracciato e index erano puliti; la guida e altri artefatti intenzionali erano untracked. Il codice descritto è committato localmente. Push e merge non sono attestati da questa guida e non sono autorizzati da questo incarico.

**Synthetic World v1 Italy, User-only functional scope: IMPLEMENTED AND LOCALLY COMMITTED.** Layer 0–1, runtime socioeconomico Layer 2, Source Simulation, Simulated System persistenti e collegamenti Source/Destination della Simulated Application sono implementati. Il runtime e l’integrazione dei sei campi socioeconomici hanno ricevuto review indipendenti; sono stati aggiunti miglioramenti mirati ai test prima dei commit.

Il completamento riguarda il perimetro funzionale User italiano. Sicurezza, certificazione memoria/risorse e indagini sulle allocazioni restano differite per decisione dell’utente; il finding Brotli noto non è dichiarato risolto. Snowflake con credenziali, pubblicazione effettiva degli asset e deployment/smoke test di produzione restano distinti dalle verifiche locali (§9).

Questa guida raccoglie obiettivi, decisioni, stato verificato e riferimenti. È documentazione di orientamento **non canonica**: non congela contratti, non modifica autorità e non certifica un rilascio. Il codice committato descrive il comportamento implementato; i contratti frozen governano le semantiche; i report attestano le verifiche delle rispettive revisioni.

## 1. Come proseguire da questa guida

Per una nuova sessione:

> Leggi gli AGENTS.md applicabili e SYNTHETIC_WORLD_GUIDA_PROGETTO.md. Verifica branch, HEAD e stato locale, poi svolgi soltanto l’incarico richiesto, rispettando i contratti frozen e le decisioni di prodotto registrate.

La lettura non autorizza automaticamente sviluppo, nuovi freeze, review generali, staging, commit o push. Non riprendere i vecchi prompt come istruzioni correnti: runtime Layer 2 e integrazione applicativa sono già conclusi. HubSpot, Companies e controlli d’identità del modello nei checkpoint non sono il prossimo incremento obbligatorio.

1. Leggere prima `AGENTS.md`, lo stato di questa guida e il §8; usare l’Appendice A per raggiungere i proprietari del codice coinvolto.
2. Distinguere comportamento implementato, vincoli normativi, prove indipendenti, verifiche differite e possibilità future. Consultare integralmente le sezioni normative pertinenti all’incarico, senza riaprire i contratti.
3. Verificare soltanto gli input rilevanti sul checkout effettivo. Le date e gli hash nei report storici non attestano automaticamente il codice corrente.
4. Attendere un incarico esplicito per il nuovo lavoro. Le possibilità del §9 non costituiscono una roadmap autorizzata.

Preferenze di collaborazione: valutare prima prodotto e usabilità, poi fattibilità tecnica; mettere in discussione le idee quando serve. Per le API dei connector privilegiare semplicità e affidabilità per chi le implementa, anche con un LLM, tenendo conto dell’efficienza. Cercare la soluzione più semplice ragionevolmente efficace, con poche iterazioni e senza conversioni o allocazioni evitabili per record.

Non occorre ricostruire tutte le conversazioni: il [riassunto storico](SYNTHETIC_WORLD_RIASSUNTO_CONTESTO.md) e il [manifest originario](SYNTHETIC_WORLD_FILE_MANIFEST.json) conservano il contesto precedente. Non descrivono lo stato corrente completo.

## 2. Che cos’è Krenalis

Secondo il [README del repository](README.md), Krenalis è una **Customer Data Platform**, cioè una piattaforma per raccogliere, trasformare, arricchire, unificare e attivare i dati di utenti e clienti.

I dati provengono da applicazioni SaaS, database, file ed eventi. La risoluzione dell’identità collega le osservazioni che appartengono alla stessa persona, producendo profili unificati da consultare e utilizzare nei sistemi operativi. L’architettura è centrata sul data warehouse del cliente: il README descrive Snowflake e PostgreSQL come archivi principali per eventi e profili, con identity resolution eseguita nel warehouse.

Il progetto è un monorepo Go con frontend TypeScript/React e API REST, distribuito come singolo eseguibile. Le dipendenze sono vendorizzate. Questa descrizione del prodotto deriva dal README; non costituisce una nuova verifica di tutte le funzionalità di Krenalis.

## 3. Che cos’è Synthetic World e perché serve

Synthetic World genera una popolazione sintetica canonica e le rappresentazioni imperfette delle sue persone in sorgenti differenti. Per la prima versione, un World rappresenta la popolazione di utenti/clienti di un’organizzazione simulata; non serve introdurre un’ulteriore entità Organization.

Krenalis ha bisogno di dati riproducibili per dimostrare e verificare ingestion, deduplicazione interna a una sorgente, risoluzione dell’identità fra sorgenti e costruzione dei profili. Una collezione di righe casuali senza verità nota non permette di stabilire se gli accoppiamenti trovati dal sistema siano corretti. Qui un Oracle separato conosce le associazioni vere.

La distinzione fondamentale è fra:

- **Person canonica:** la verità generata per una persona.
- **SourceRecord:** una sua osservazione in una sorgente, eventualmente incompleta o discordante.
- **Oracle:** la relazione nascosta che permette di confrontare il risultato di Krenalis con la verità.

Per esempio, P42 può essere assente dalla sorgente A, comparire due volte nella B e una volta nella C. I due record della B possono avere nomi rappresentati diversamente, telefoni mancanti o indirizzi differenti. Tutti e tre i record effettivamente prodotti restano associati a P42 nell’Oracle, senza esporre tale chiave a Krenalis attraverso gli output ordinari.

L’obiettivo statistico italiano è **Demo Plausible**: dati coerenti, vari e credibili per questo uso. Le qualificazioni esistenti non attestano idoneità a inferenze statistiche sulla popolazione reale. L’obiettivo ingegneristico resta rigoroso: determinismo, identità corrette, separazione delle responsabilità e uso controllato delle risorse.

La priorità esplicita dell’utente è **la soluzione più semplice ragionevolmente efficace, con poche iterazioni**. Fissare ora soltanto ciò che sarebbe costoso riprogettare; lasciare modificabili i dettagli e rinviare le capacità senza un caso d’uso attuale. Non introdurre nuovi contratti o nuove review generali per ogni scelta locale d’implementazione.

## 4. Stato attuale dei dati e del runtime

| Parte | Stato corrente | Significato concreto |
| --- | --- | --- |
| Layer 0 | IMPLEMENTATO; baseline frozen | Identità, coordinate, componenti e stream deterministici in `tools/fakedata` |
| Layer 1 — Italia | IMPLEMENTATO; baseline frozen | Person canonica, geografia, nomi, contatti, catalogo foto e semantiche World esistenti |
| Contratti Layer 2 / D2-A | FROZEN | Semantiche socioeconomiche, dipendenze, rappresentazioni e limiti normativi |
| Protocollo deterministico / loading amendment | FROZEN e revisionati | Byte, fingerprint, campionamento e acquisizione esterna separata dalla costruzione |
| Modello italiano it-2026a | Prodotto e qualificato come asset | Modello logico canonico; disponibilità sulle origin di deployment da verificare separatamente |
| Runtime Go Layer 2 | IMPLEMENTATO e revisionato indipendentemente | `SocioeconomicWorld`, Company, Employer, Education, attività, occupazione e sei rami Income |
| Estensibilità / Source Simulation | Contratto frozen; runtime IMPLEMENTATO | Osservazioni, duplicati, ID locali opachi, Oracle separato e adapter CSV |
| Workspace Environment | IMPLEMENTATO | Production/Development immutabile; connector reali disponibili anche in Development |
| Simulated System / Admin Simulated world | IMPLEMENTATO e committato | Dati persistenti per workspace, generazione/ripresa, CRUD e progresso |
| Simulated Application Source/Destination | IMPLEMENTATA | Collegamenti, lettura e scrittura persistenti, ammissione dei writer e aggiornamenti parziali |
| Sei proprietà socioeconomiche Source | IMPLEMENTATE e revisionate indipendentemente | Generazione `generation-v4`; compatibilità `generation-v3` ed `empty-v1` |

### Milestone locali verificate nel log Git

| Commit | Lavoro concluso |
| --- | --- |
| `9300b0e9bcaba35e328dc3723c36d5ac3d283357` | Freeze del protocollo deterministico Layer 2 e dell’external-model-loading amendment |
| `fef655b7e5b42d04c4148b5f92e2f05818ed7307` | Runtime socioeconomico deterministico Layer 2 |
| `50ed99476cf22a8137d1adcf8e2c82f7235e0ec6` | Integrazione dei sei campi socioeconomici nei Simulated System |

I commit precedenti `bce4ff958` e `1a01600d3` introducono rispettivamente Simulated System/Simulated World ed External Assets; `e3ca1442a` aggiunge la fondazione del modello. Le descrizioni “working tree non committato” nei report si riferiscono alle loro sessioni, prima di questi commit.

### Layer 0–1: codice già funzionante

[layer1.go](tools/fakedata/layer1.go) definisce `Person`, `WorldConfig`, `World` e `World.Person(index)`. Person contiene ID, genere, data di nascita, età corrente, nome e cognome, email, telefono, sito, indirizzo e PhotoID. `Address.Country` è un valore dell’indirizzo italiano; non va reinterpretato automaticamente come cittadinanza o come soluzione universale del contesto paese.

WorldConfig contiene namespace delle identità, seed, data di riferimento e catalogo foto. Il runtime attuale è italiano. La costruzione richiede un catalogo foto valido, con la copertura richiesta; l’adattamento futuro deve usare questo punto di ingresso senza rimuovere il requisito.

Esiste una [fixture selezionata di foto](tools/fakedata/testdata/fakefacegen-v1-selected/FIXTURE.md), usata dai [test Layer 1](tools/fakedata/layer1_test.go) attraverso il caricamento dedicato ai record selezionati. Contiene gli asset di due ID e **non è un catalogo fakefacegen completo**. È un punto di partenza per i test, non un catalogo di produzione da passare indistintamente a ogni loader. `PhotoID` è locale al catalogo e non è una chiave d’identità della persona.

La generazione è ad accesso diretto: non occorre generare tutte le persone precedenti. Il dominio operativo di Layer 1 è **1..60.466.176**. L’indipendenza dalla dimensione della popolazione vale entro il dominio ammesso, non significa indici illimitati.

[identity.go](tools/fakedata/identity.go) deriva PersonID dal namespace e dall’indice, senza includere il seed del World. Cambiare seed può cambiare i valori senza cambiare quella identità. [stream.go](tools/fakedata/stream.go) e [component.go](tools/fakedata/component.go) forniscono indirizzamento deterministico per entità, percorso e dipendenze. Sono primitive esistenti, in parte private; il runtime Source Simulation aggiunto successivamente le affianca in `source_simulation.go`. WorldID e SnapshotID hanno significati distinti: il secondo include anche data di riferimento e catalogo foto; questa separazione è frozen.

[layer1_data.go](tools/fakedata/layer1_data.go) incorpora sei CSV sotto `tools/fakedata/data/it-2026a/`: pesi delle età, nomi, cognomi, località, codici postali e nomi delle strade. Il caricamento usa `sync.OnceValues` e verifica gli hash previsti. I file di provenienza e le fixture già presenti restano separati dal nuovo lavoro.

### Layer 2: runtime socioeconomico italiano implementato

[SocioeconomicWorld](tools/fakedata/socioeconomic.go) avvolge il World base autorevole e un [SocioeconomicModel](tools/fakedata/socioeconomic_model.go) acquisito, opaco e immutabile. Offre accesso diretto a `Company`, `Facts`, `Person` composta e `ID`; non modifica le identità e le semantiche Layer 0–1. `Facts(index)` usa età all’ancoraggio e regione senza generare foto, contatti o una Person completa.

Il runtime genera l’universo Company e costruisce quaranta indici Employer. Il modello logico italiano it-2026a prevede 65.536 Companies. `Company` contiene ID, numero modellato di addetti, Industry e SizeBand; gli addetti modellati non sono il conteggio delle persone materializzate o assegnate. Le aziende con zero addetti esistono ma non sono datori eleggibili. Employer seleziona una Company per il dipendente secondo i pesi frozen, senza popolazione globale o occupancy.

[SocioeconomicFacts](tools/fakedata/socioeconomic_types.go) contiene Education, PrimaryActivity, payload alternativi Employee/SelfEmployed e Income. Education è il titolo/livello più alto **completato**, non l’iscrizione corrente. Employee riferisce Employer tramite `CompanyIndex` e possiede JobFamily/JobLevel; la sua Industry proviene dalla Company scelta. SelfEmployed possiede Industry/JobFamily e non ha Employer o JobLevel. Gli altri rami non hanno payload occupazionale.

La data di ancoraggio è `2026-01-01`. Cambiare ReferenceDate può cambiare CurrentAge, senza far evolvere i fatti socioeconomici. I soli fatti Layer 1 ammessi nel DAG sono età all’ancoraggio e regione, con i vincoli specifici dei nodi; genere, foto, età corrente e data di riferimento non diventano dipendenze socioeconomiche.

[La generazione dei fatti](tools/fakedata/socioeconomic_facts.go) implementa i sei rami Income: Employee, SelfEmployed, Unemployed, Student, Retired e Inactive. Income è reddito personale monetario lordo annualizzato alle condizioni dell’ancoraggio, in euro interi (`AnnualIncomeEUR uint32`, 0..4.294.967.295). Non è automaticamente stipendio, reddito familiare, fatturato aziendale o importo realizzato in un anno storico. Zero è un reddito numerico reale, non missingness. Il runtime campiona il valore finale dalle leggi finite compilate; conversioni e aritmetica compatibili seguono i contratti interi/razionali esatti, senza floating point.

I JSON D2-B2 locali e i report di produzione conservano modello canonico, pubblicazione, crosswalk ed evidenza del produttore offline. La [review canonica finale](SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_ASTRA_CANONICAL_ASSET_REVIEW.md) attesta **PASS — D2-B2 v3 FROZEN**, con 14/14 controlli numerici HARD e osservazioni separate. Questa qualificazione non dimostra che l’asset sia raggiungibile su ogni origin di produzione. I grandi documenti di evidenza non sono input per generare ogni persona.

## 5. Contratti Layer 2 e acquisizione del modello

| Riferimento frozen | Ambito |
| --- | --- |
| [Contract 1 v1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md) | API, World wrapper, identità e attività principale |
| [Contract 2 v1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md) | Universo Company, addetti modellati e Employer |
| [Contract 3 v1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md) | Tassonomie, occupazione e applicabilità |
| [Contract 4 v1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md) | DAG e isolamento delle dipendenze |
| [Contract 5 v1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md) | Income, rappresentazione e scheletro composto |
| [D2-A v1](SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md) | Architettura numerica, validazione e limiti runtime |
| [Protocollo deterministico v1](SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md) e [vettori](SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_VECTORS_v1.json) | Realizzazione deterministica `socioeconomic-v1`, framing, identità, fingerprint e campionamento |
| [External model loading amendment v1](SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md) | Acquisizione, ownership e trasferimento esplicito del modello alla costruzione |

Contracts 1–5 e D2-A restano superiori nelle rispettive semantiche. L’amendment sostituisce **soltanto** le clausole elencate nel suo §2: il vecchio costruttore con solo World e l’acquisizione embedded non descrivono l’API corrente. Il protocollo completa le scelte deterministiche lasciate aperte senza cambiare il modello numerico. Non sostituire Contract 5 v1 FROZEN con le versioni v0 review-ready/stabilized, né interpretare le intestazioni storiche “proposed” come revoca del freeze.

### 5.1 Tre fasi distinte, già implementate

```text
External Assets → Load(ctx) → modello validato/compilato immutabile
    → NewSocioeconomicWorld(base, model) → Company e indici Employer
    → Company / Facts / Person → generazione deterministica locale
```

Il percorso di produzione è fisso: **`synthetic-world/it-2026a/model.json`**. [cmd](cmd/server.go) costruisce `SocioeconomicModelSource` dalle origin ordinate di `KRENALIS_EXTERNAL_ASSETS_URLS` e lo passa a Core tramite `SimulatedSocioeconomicSource`. Il default configurato è `https://assets.krenalis.com/`; il [resolver esistente](tools/externalassets/resolver.go) gestisce il fallback. Non esiste una seconda configurazione per il modello. `KRENALIS_SYNTHETIC_PHOTOS_DIR` riguarda le foto, non un percorso alternativo del modello.

`Load(ctx)` acquisisce e valida il modello completo, compila lo stato privato e prepara digest/fingerprint prima di pubblicarlo. Conserva gli esiti riusciti per il riuso, coordina le chiamate concorrenti e permette nuovi tentativi dopo fallimenti. Il modello non è embedded né generato in Go. Il JSON di produzione resta esterno a Git; la piccola [fixture applicativa](core/testdata/socioeconomic-model.json) è artificiale e non ne attesta il deployment.

Costruzione e generazione non scaricano né riscoprono asset. Il World conserva base e modello immutabili; può usarli dopo che il source o l’origine non sono più disponibili. Più Worlds possono condividere il modello. Il worker carica il modello soltanto per preparazioni v4; startup ordinario, letture Source, aggiornamenti Destination, v3 ed empty-v1 non avviano quell’acquisizione. Una preparazione v4 già persistita può naturalmente attivarla dopo il riavvio.

Le identità dipendono dal contenuto logico canonico e dagli input frozen, non da origin, compressione o formattazione JSON. Gli hash della publication, dei report o dell’intero bundle non sono seed di generazione. [Identity](tools/fakedata/socioeconomic_identity.go) e [sampling](tools/fakedata/socioeconomic_sampling.go) realizzano il protocollo; il costruttore riceve `base *World, model *SocioeconomicModel`.

### 5.2 Politica approvata di pubblicazione immutabile

**Il modello socioeconomico logico it-2026a è immutabile una volta pubblicato.** Tutte le origin External Assets configurate devono fornire contenuto logicamente equivalente per questa release. Un modello diverso richiede una nuova release esplicita, non la sostituzione del contenuto logico allo stesso percorso.

Questa è una politica operativa approvata, non una garanzia automaticamente imposta dal checkpoint. L’applicazione **non persiste SocioeconomicWorldID nei checkpoint** e non rileva così una sostituzione non autorizzata fra processi. L’omissione è intenzionale e approvata; aggiungere tale guard non è lavoro richiesto. Cache di processo, validazione del modello e identità deterministiche non certificano l’immutabilità delle origin fra riavvii. La disponibilità del vero asset su tutte le origin configurate non è stata verificata da questo aggiornamento.

## 6. Architettura approvata per estensibilità e sorgenti

Il documento di riferimento è [SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0.md](SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0.md). La [review Astra](SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0_ASTRA_REVIEW.md) ha verificato stato del repository e sufficienza dei confini, con **6/6 controlli concettuali PASS, 0 BLOCKER, 1 NON_BLOCKING e 1 OBSERVATION**.

È una review architetturale eseguita prima del codice Go delle sorgenti, distinta dalle verifiche del runtime. Il contratto originale Source Simulation non è il successivo protocollo deterministico Layer 2 e non ne governa stream o caricamento. Il freeze è stato registrato senza cambiare il contratto in [SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0_FREEZE_RECORD.md](SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0_FREEZE_RECORD.md). L’intestazione DRAFT del contratto è storica. Le implementazioni successive hanno proprie verifiche, distinte dai sei controlli concettuali.

I sei controlli sono già stati valutati concettualmente:

| Controllo | Esito | Proprietà verificata |
| --- | --- | --- |
| A — Italia esistente | PASS | Adattamento del modello italiano senza cambiarne le semantiche frozen |
| B — Secondo paese | PASS | Nuovi dati/generatori e modifiche locali, senza ridisegnare sorgenti e Oracle |
| C — Nuovo campo canonico | PASS | Modifiche locali a tipi e mappature, preservando membership, molteplicità e Oracle |
| D — Sovrapposizione parziale | PASS | Popolazioni condivise e distinte, senza obbligo di indipendenza statistica |
| E — Duplicato nella stessa sorgente | PASS | Record distinti e anche discordanti, con la stessa persona vera |
| F — Indipendenza dall’ordine | PASS | Risultati logici invariati per richieste equivalenti riordinate o in batch |

Il NON_BLOCKING riguarda soltanto la sequenza implementativa del contratto originale. L’OBSERVATION considera innocuamente più restrittiva del necessario la scelta di rifiutare un paese indisponibile prima della generazione: non chiede di modificarla. **Nessuno dei due rilievi richiede una rettifica semantica del contratto approvato.**

```text
World: popolazione canonica e configurazione
    ↓ scelta deterministica del contesto paese
Generazione specifica del paese: IT oggi, altri paesi in futuro
    ↓
Persona canonica + contesto paese
    ↓ per ciascuna SourceInstance
Membership → molteplicità → identità locali e osservazioni
    ↓
0..N SourceRecords per persona e sorgente
    ├── CSV / documenti della Simulated Application → Krenalis
    └── mapping Oracle separato → valutazione
```

### Paesi e nuovi campi

Il World concettuale possiede configurazione, coordinate delle persone e scelta del paese. Può essere un contenitore esterno all’attuale `fakedata.World` italiano. Un confine di generazione isola dati e logica specifici del paese; `CountryPack` è un nome di lavoro, non un’interfaccia Go obbligatoria.

Ogni persona ha un contesto paese univoco, disponibile dove serve. È il paese della generazione geografica/modellistica, non necessariamente nazionalità o cittadinanza. Il contesto può vivere accanto alla Person: non è richiesto aggiungere `Country` allo schema frozen.

La composizione del World usa una politica di selezione relativa per persona, **non quote finite esatte**. A parità di configurazione, seed/versione e PersonID, il paese non cambia con ordine, concorrenza, sottoinsieme richiesto o numero di persone materializzate. La dimensione della popolazione non entra nell’assegnazione. Un paese configurato ma non disponibile provoca un errore esplicito prima della generazione, senza sostituzione silenziosa.

Un secondo paese richiede nuovi dati e codice di generazione, ed eventualmente mappature locali. Non richiede di ridefinire membership, molteplicità, identità dei record, Oracle o architettura dei trasporti. Le classificazioni private italiane non diventano requisiti del motore generico.

Il confine approvato permette in futuro persone di più paesi nella stessa sorgente; il runtime disponibile oggi è italiano. Le osservazioni che richiedono formattazioni o trasformazioni nazionali possono usare il contesto paese e delegare al generatore specifico o alla mappatura locale del campo, senza inserire regole italiane in membership, molteplicità o Oracle.

I fatti canonici restano tipizzati. Un nuovo campo, per esempio `PreferredLanguage`, può richiedere modifiche locali a tipi, generatori, osservazioni e adapter. Non cambia le regole che associano persone, sorgenti e record. Non serve un property bag o un motore universale di schema.

### Membership, duplicati e osservazioni

La cardinalità è **`(Person, SourceInstance) → 0..N SourceRecords`**. Non è una relazione 0/1 e non ha un massimo semantico fissato. La politica iniziale implementata usa casi 0/1/2; non è un massimo semantico imposto alle evoluzioni future.

Le sorgenti hanno decisioni di membership separate. Questo consente persone in nessuna sorgente, in una sola, in alcune o in tutte; non impone indipendenza statistica. Coverage semplici bastano inizialmente e non garantiscono percentuali esatte in ogni campione. Una futura politica di correlazione potrà cambiare le scelte senza ridefinire record o Oracle.

Ogni record ha un’identità locale e osservazioni proprie. Può anche conservare correttamente i valori canonici: degradarli non è obbligatorio. Due record della stessa persona possono differire per campi presenti, valori mancanti, rappresentazioni e valori discordanti. Persona assente dalla sorgente e campo assente in un record sono condizioni distinte. Nessuna causa del duplicato o storia temporale è richiesta.

### Determinismo, identità e Oracle

Gli input semantici stabili comprendono World/configurazione/seed/versioni, identità e configurazione della sorgente, PersonID, discriminante stabile del record e discriminanti di campo/operazione quando necessari. Devono determinare esistenza, numero, identità e osservazioni dei record. Iterazione delle persone o sorgenti, scheduling, mappe Go, batch e richieste estranee non sono input nascosti. Non si usa un unico generatore casuale condiviso consumato nell’ordine di visita.

Il contratto Source Simulation non promette stabilità degli output fra configurazioni World/source deliberatamente diverse o versioni future del modello. Il determinismo vale a parità di input semantici; restano comunque intatte le garanzie proprie dei Layer 0–2 frozen.

SourceRecordID è stabile e univoco nella propria SourceInstance. Gli output ordinari non espongono PersonID, mapping Oracle, identificatori di gruppi duplicati o una chiave canonica comune che renda banale il matching. Valori osservati coincidenti, come un’email, restano leciti. L’obiettivo è evitare la divulgazione accidentale della verità, non costruire un protocollo di segretezza crittografica.

Per v1 ogni SourceRecord generato ha esattamente una persona vera. L’Oracle è contenuto nel World/contesto di valutazione e mantiene:

```text
(SourceInstanceID, SourceRecordID) → PersonID
```

Più chiavi, anche della stessa sorgente, possono puntare alla stessa persona. Questo basta per valutare deduplicazione interna e matching fra sorgenti. Identiche chiavi locali in World differenti appartengono a Oracle differenti; non serve aggiungere WorldID a ogni chiave se il contenimento resta esplicito. Il percorso Oracle rimane separato dagli output e dalle interfacce ordinarie di ingestion.

### Trasporti e memoria

Il generatore decide membership, quantità, osservazioni, identità locale e verità. CSV e future API rappresentano quei record: non ridisegnano o riestraggono tali decisioni. Codifiche diverse degli adapter non cambiano l’associazione Oracle.

La generazione resta possibile per persona o in batch limitati, senza popolazione completa, shuffle globale, tabella globale di deduplicazione o collezione completa dei record in RAM. Tabelle immutabili e indici del modello non equivalgono a materializzare la popolazione.

## 7. Confini ingegneristici per futuri incarichi

Le implementazioni concrete esistono: non trattare API, loader, persistenza o adapter già presenti come decisioni ancora da prendere. Le possibilità d’estensione restano nei limiti dei contratti frozen e dell’incarico effettivo.

- Cercare prima utility interne, standard library e dipendenze vendorizzate; evitare nuove dipendenze senza necessità concreta.
- Usare l’astrazione JSON del repository, [tools/json](tools/json/functions.go), per il lavoro nuovo. Non riscrivere codice frozen per uniformità.
- Rispettare `AGENTS.md`, validazione, limiti e confini di fiducia. Le verifiche funzionali concluse non sostituiscono la certificazione risorse differita.
- Conservare separati fatti canonici tipizzati, osservazioni, trasporto e Oracle. Un nuovo campo richiede modifiche locali appropriate, non un motore universale di schema.

[types.MarshalValidate](tools/types/value.go) **è implementata** con firma `func MarshalValidate(data any, schema Type) (json.Value, error)`: valida la rappresentazione Go contro lo schema e serializza, restituendo `*SchemaValidationError` per dati non conformi. Il percorso Destination corrente la usa. La discussione di settembre sulla firma e sui buffer è conclusa rispetto al codice attuale; non è un punto di ripartenza aperto. Non è rivendicata una certificazione delle allocazioni.

## 8. Simulated System e Simulated Application correnti

### 8.1 World implicito e sistemi indipendenti

Il World canonico è interno e derivato stabilmente dall’ID del workspace; l’utente non crea o configura un World. Un Simulated System appartiene a un solo workspace Development e conserva i propri record nel warehouse di quel workspace. Sistemi distinti hanno stati indipendenti, anche quando selezionano alcune delle stesse persone canoniche. Creare o rinominare un sistema non ridefinisce le persone degli altri sistemi.

Un sistema non è più un account di uno specifico provider S3 o HubSpot. Il connector dedicato è **Simulated Application**, codice `simulated-application`, registrato in [core/internal/simulated](core/internal/simulated/application.go) per **User Source e User Destination server-side**. I connector reali continuano a funzionare anche in Development; Environment non implica intercettazione delle chiamate né assenza di operazioni su dati reali.

Ogni sistema può avere **al massimo una connection Source e una Destination**, entrambe nello stesso workspace. Il binding persistente è `connections.simulated_system`, con foreign key composta per workspace e unicità per sistema/ruolo. Il selettore ammette sistemi Available senza connection già presente per quel ruolo. Una connection esistente non può essere riassegnata a un altro sistema. Questi controlli sono nel backend, oltre che nell’UI.

### 8.2 Configurazione, API e persistenza

L’API Admin è **`/v1/simulated/systems`**: POST crea, GET elenca/legge, PUT rinomina, DELETE avvia l’eliminazione. Usa sessione Admin e workspace autorizzato; non accetta autenticazione API key. Creazione e utilizzo simulato richiedono Development. `environment` è `production` (default) o `development`, selezionabile alla creazione del workspace e immutabile.

| Configurazione corrente | Semantica e limiti |
| --- | --- |
| `name` | Nome validato, massimo 100 caratteri; unico attributo iniziale rinominabile |
| `userCount` | Intero 0..10.000.000; numero esatto di record iniziali, duplicati inclusi |
| `duplicateRecordPercent` | 0..50%, fino a due decimali; zero obbligatorio per sistema vuoto |
| `countries` | Oggetto ISO alpha-2 → peso relativo intero 1..100; soltanto IT; richiesto per conteggio positivo |
| Configurazione iniziale | Immutabile; non esistono seed o namespace configurabili dall’utente |

`userCount` non è il numero di persone canoniche né il conteggio corrente dopo Destination. Il limite di generazione iniziale non è una certificazione di capacità né un limite dichiarato alla successiva crescita. I pesi paese non devono sommare a 100; oggi c’è soltanto Italy, senza runtime multinazionale o quote multinazionali implementate.

Nel PostgreSQL operativo, [simulated_systems](core/internal/initdb/schema.sql) conserva ID, workspace, nome, stato, configurazione, policy, conteggio generato, checkpoint JSON, errore, timestamp e autorità dei writer (`writer_generation`, `claimed_run`, `claim_ping_time`). Non ha una colonna connector. World, modello e Person completi non sono persistiti nei metadata.

Nel warehouse, [PostgreSQL](warehouses/postgresql/tables/simulated_system_records.sql) e [Snowflake](warehouses/snowflake/tables/simulated_system_records.sql) definiscono `krenalis_simulated_system_records` con `simulated_system`, `external_id`, `updated_at` e `data` (JSONB/VARIANT). I record sono identificati dalla coppia sistema/ID osservato. Metadata, checkpoint e mapping Oracle non sono inseriti nel documento `data`; il datastore risolto per workspace mantiene il confine d’isolamento. L’implementazione non usa la vecchia tabella simulated-account a tre colonne.

### 8.3 Generazione, stati e ripresa

| Stato | Comportamento |
| --- | --- |
| `Preparing` | Generazione in corso o riprendibile; non eliminabile né selezionabile per nuove connection |
| `Available` | Preparazione conclusa; oppure sistema vuoto immediatamente disponibile |
| `Failed` | Preparazione fallita; nessuna rigenerazione/retry automatico del sistema terminale |
| `Deleting` | Intento di eliminazione persistito; pulizia asincrona di record e metadata |

Zero record usa `empty-v1`, diventa Available senza checkpoint o caricamento foto/modello. Un nuovo sistema non vuoto usa **`generation-v4`** e parte Preparing. I sistemi storici **`generation-v3`** mantengono generazione e ripresa precedenti senza acquisizione Layer 2. I sistemi Available non vengono arricchiti retroattivamente o rigenerati.

Il [worker corrente](core/simulated_system_generation.go) appartiene a Core; soltanto il leader elabora sistemi Preparing/Deleting, ordinati per creazione e ID, in serie, con wake-up e scansione ogni 5 secondi. La preparazione non vuota richiede un catalogo foto valido; v4 acquisisce inoltre il modello e costruisce un wrapper socioeconomico sullo **stesso World base** usato da SourceWorld.

Se `N=userCount` e `p` è la percentuale in centesimi, `D=floor(N*p/10000)` sono i record aggiuntivi: si selezionano `N-D` persone, delle quali `D` con due osservazioni. Cento record al 10% rappresentano novanta persone. La selezione usa la permutazione affine del dominio Layer 1 specifica del sistema. Il World deriva dal workspace e usa la data di riferimento fissa `2026-01-01`; il nome del sistema non modifica questi input.

Il worker genera batch di **massimo 256 record**. Esegue Merge nel warehouse, poi salva checkpoint e conteggio assoluto nel PostgreSQL operativo. Il checkpoint contiene cursore persona/rappresentazione, identità del catalogo foto, versioni base/source/namespace, data di riferimento e URL foto. Non si ricostruisce il cursore contando righe. Il replay dopo Merge senza checkpoint rigenera gli stessi ID e documenti a parità di input; `updated_at` resta metadata wall-clock e può avanzare. Prima di Available si verifica il totale nel warehouse e si chiude l’autorità di generazione.

Errori permanenti di acquisizione/configurazione/costruzione/generazione/conteggio portano al percorso Failed; errori warehouse transitori, cancellazione, arresto o perdita della leadership lasciano la preparazione riprendibile. Un fallimento di acquisizione resta ritentabile dal source per un altro job ammissibile, senza riavviare automaticamente un sistema Failed. Non viene usato un fallback a record Layer 1 per fingere il completamento v4.

La ripresa socioeconomica si fonda sulla politica del modello immutabile (§5.2), **senza confronto di SocioeconomicWorldID nel checkpoint**. Le protezioni attuali di scrittura e fencing non sono una transazione atomica fra i due database.

### 8.4 Source, Destination e concorrenza

[Connections](core/internal/connections/connections.go) inietta lo store simulato soltanto nel connector dedicato. La Source legge documenti persistenti, li decodifica contro lo schema Source completo e verifica `customer_id == external_id`; il confine [Application](core/internal/connections/application.go) proietta poi lo schema richiesto dalla pipeline. Letture paginate usano timestamp/cursore, con pagine fino a 1.000 record. I fatti non vengono ricalcolati alla lettura.

Destination crea nuovi record con ID osservato locale oppure applica patch ai record esistenti. [Il mapper Destination](core/internal/simulated/application_destination.go) legge il documento completo, valida le proprietà scrivibili, fonde la patch in Go e riscrive il documento tramite datastore; le proprietà non incluse nella patch restano conservate. L’indirizzo viene aggiornato per proprietà, senza un motore di patch SQL specifico per warehouse.

L’ammissione centrale dei run consente un solo writer Destination attivo per sistema. Claim persistenti, generazioni dei writer e fencing nel datastore proteggono il percorso, con recupero dei writer interrotti. I riferimenti sono [Core](core/core.go), [gestione writer](core/simulated_system_writers.go) e [scritture datastore](core/internal/datastore/simulated_records_write.go).

Cancellare una connection lascia il sistema. L’API rifiuta l’eliminazione di un sistema ancora referenziato; l’Admin può coordinare prima l’eliminazione delle connection e poi quella del sistema, mostrando i passaggi. La pulizia asincrona attende che non restino writer ammessi e applica fencing prima di rimuovere i dati.

### 8.5 Le sei proprietà socioeconomiche Source

La [Simulated Application Source](core/internal/simulated/application_schema_source.go) espone esattamente queste **sei aggiunte socioeconomiche**, oltre alle normali proprietà osservate:

| Proprietà | Origine canonica e applicabilità nei nuovi record v4 |
| --- | --- |
| `education_level` | `Facts.Education`; tutte le Persons generate |
| `primary_activity` | `Facts.PrimaryActivity`; tutte; `employee`, `self-employed`, `unemployed`, `student`, `retired`, `inactive` |
| `annual_income_eur` | `Facts.Income`; tutte; numero intero unsigned a 32 bit in EUR, incluso zero |
| `job_family` | Employee.JobFamily oppure SelfEmployed.JobFamily; soltanto Employee/SelfEmployed |
| `industry` | Industry della Company selezionata per Employee; SelfEmployed.Industry per SelfEmployed |
| `job_level` | Employee.JobLevel; soltanto Employee |

Per Unemployed, Student, Retired e Inactive i tre campi occupazionali sono **assenti**. Per SelfEmployed è assente JobLevel. Non si usano null, stringhe vuote o ordinali come sostituti dell’inapplicabilità. Income zero è serializzato come numero `0`, preservando la distinzione dall’assenza.

I sei campi sono opzionali in lettura e non nullable: record legacy e record creati da Destination possono ometterli. Lo [schema Destination](core/internal/simulated/application_schema_destination.go) non li include e il validatore ne rifiuta la scrittura, sia in creazione sia in aggiornamento. Una patch alle proprietà ordinarie conserva i campi socioeconomici esistenti decodificando e riserializzando con lo schema Source completo.

[La materializzazione](core/simulated_system_document.go) usa `Facts` sul PersonIndex canonico effettivamente selezionato. Le osservazioni duplicate condividono i fatti socioeconomici, mantenendo ID osservati distinti e le proprie variazioni ordinarie. Anche sistemi diversi che osservano la stessa Person condividono quei fatti a parità di World/modello; gli stati persistenti restano indipendenti.

Non vengono esposti Employer/Company ID, CompanyIndex o record Company. Nessuna proprietà è aggiunta automaticamente a HubSpot o agli altri provider: questo è lo schema della **Simulated Application**. Il CSV Source Simulation conserva il proprio schema e non riceve implicitamente le sei aggiunte.

### 8.6 Admin Simulated world

[SimulatedWorld.tsx](admin/src/components/routes/SimulatedWorld/SimulatedWorld.tsx) espone elenco, creazione, dettaglio, progresso reale, rinomina ed eliminazione nei workspace Development. I controlli correnti comprendono conteggi predefiniti da zero a 10.000.000 e Custom, percentuali duplicati predefinite/Custom, Italy e creazione opzionale delle connection Source e Destination.

L’UI crea prima il sistema, attende Available mostrando il progresso, poi crea le connection richieste con `settings: { account: id }`. Se una connection fallisce, il sistema già creato resta disponibile. Gli errori di polling vengono ritentati senza inventare uno stato Failed. Il polling della preparazione/eliminazione è ogni 3 secondi; quello ordinario dei metadata ogni 15 secondi. Il dettaglio mostra Italy, non una percentuale dedotta dal peso paese. Non sono presenti controlli socioeconomici, selezione del modello o browser dei fatti.

### 8.7 Decisione di prodotto: Companies e Groups differiti

**Conservare Company ed Employer nel runtime; non esporne ancora relazioni o record nella Simulated Application.** Krenalis si concentra oggi sulle entità User e non offre ancora Groups come capacità di prodotto richiesta per questo uso. L’utente vuole rivalutare Companies quando saranno disponibili Groups o un modello organizzativo adatto.

È una funzionalità di prodotto deliberatamente differita, non un difetto né un requisito mancante per completare Layer 2. Non aggiungere ora `employer_id`, `company_id` o record Company. Un eventuale Group futuro non è automaticamente equivalente a una Employer Company: quella corrispondenza richiederà una decisione esplicita.

## 9. Evidenze, verifiche differite e possibilità future

### 9.1 Completato e revisionato indipendentemente

| Evidenza | Cosa attesta e come leggerla oggi |
| --- | --- |
| [Technical review del protocollo](SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_FULL_TECHNICAL_REVIEW.md) | Compatibilità deterministica, vettori e fattibilità del candidato prima del freeze; non certificazione delle allocazioni runtime |
| [Implementation report runtime](SYNTHETIC_WORLD_V1_LAYER2_SOCIOECONOMIC_RUNTIME_IMPLEMENTATION_REPORT.md) | Consegna funzionale Layer 2 prima di review e commit; i suoi “review da svolgere” e “nessun commit” sono storici |
| [Independent runtime review](SYNTHETIC_WORLD_V1_LAYER2_SOCIOECONOMIC_RUNTIME_INDEPENDENT_CODE_REVIEW.md) | Review funzionale/deterministica positiva con rilievi non bloccanti sui test; oracle indipendente, modello reale locale, Company/Employer, sei rami Income e concorrenza |
| [Reconnaissance integrazione](SYNTHETIC_WORLD_V1_LAYER2_SIMULATED_SYSTEM_INTEGRATION_RECONNAISSANCE.md) | Analisi storica del confine applicativo; proposta Education-only e guard del modello **superate** dalle decisioni approvate |
| [Independent six-field review](SYNTHETIC_WORLD_V1_LAYER2_SIMULATED_SYSTEM_SIX_FIELD_INTEGRATION_INDEPENDENT_CODE_REVIEW.md) | Integrazione dei sei campi, compatibilità v3/v4/empty, duplicati, ripresa e ciclo reale PostgreSQL Source → import → Destination → Source; nessun difetto funzionale bloccante rilevato |

Il runtime ha ricevuto verifica indipendente dei vettori e del modello reale: 65.536 Companies, quaranta indici Employer e 2.052 Persons, con tutti i rami Income; regressione Layer 1 su 12.312 Persons. La review applicativa ha confrontato 1.024 documenti con aspettative indipendenti, inclusi duplicati, omissioni e Income zero/massimo. Sono evidenze per quei campioni e revisioni, non un’enumerazione di tutti i seed.

Le suite funzionali, le verifiche mirate di concorrenza/race e i percorsi PostgreSQL sono attestati nei report. I commit includono miglioramenti permanenti ai test, fra cui lookup delle leggi dei vettori e cancellazione/retry del proprietario dell’acquisizione nel runtime, e il [round trip Destination di un record v4 realmente generato](cmd/simulated_systems_socioeconomic_test.go) nell’integrazione. Il report six-field distingue il test `-race` del processo padre dall’instrumentazione del server esterno usata nelle sue verifiche temporanee; non dedurne che ogni esecuzione del normale harness certifichi le race del worker.

Per questo aggiornamento sono stati letti guida completa, codice/test pertinenti, contratti e report, e verificati i tre commit. **Non sono stati rieseguiti test Go, race, database, browser o qualificazioni numeriche.** Il README è stato consultato per il contesto CDP; l’handoff operativo qui descritto è stato confrontato con il codice corrente, senza modificare il README.

### 9.2 Differito per decisione esplicita dell’utente

- Security review e hardening.
- Certificazione memoria e budget delle risorse, incluse capacità e picchi di acquisizione/costruzione.
- Indagini su allocazioni e memoria di decompressione.
- Finding preesistente sulle allocazioni Brotli: riconosciuto, **non reinvestigato e non dichiarato corretto**.

I limiti normativi frozen rimangono applicabili. La review funzionale non li modifica e non equivale alla loro certificazione; questi temi non vengono riaperti da questa guida.

### 9.3 Follow-up di verifica/rilascio non interamente attestati

- Snowflake dipendente da credenziali non è stato eseguito nella recente review d’integrazione; PostgreSQL sì. Non attribuire a Snowflake l’esito delle prove PostgreSQL.
- Accettazione locale del modello reale e fixture/loader ermetici non provano la pubblicazione del vero asset immutabile su tutte le origin configurate.
- Deployment effettivo e smoke test di produzione non sono attestati dalle evidenze consultate.
- Push/merge del branch locale non sono autorizzati da questo incarico; nessuno è stato eseguito per aggiornare la guida.

### 9.4 Evoluzione di prodotto possibile, senza roadmap obbligatoria

Groups e Company/Employer applicativi seguono la decisione del §8.7. Altri paesi richiedono modelli e generatori dedicati. Simulazioni provider più realistiche, migliore varietà delle osservazioni/foto e ulteriori workflow di valutazione dell’identity resolution potranno essere considerati se utili e approvati. Non sono prerequisiti del perimetro User italiano completato.

La prova storica A+B usa ora CSV temporanei e pipeline FileSystem/CSV reali. I risultati dei vecchi report (40/20 identità, 47 profili e specifiche coppie Oracle) descrivono quella fixture, non garantiscono qualità generale. L’Oracle resta separato: un record Destination nuovo non acquisisce automaticamente PersonID o verità nota; non equivale a NO_MATCH e non va classificato ricavando verità dai profili della CDP.

## 10. Autorità e manutenzione della guida

1. **Comportamento implementato:** codice/test committati al checkout verificato; i report non lo sostituiscono.
2. **Semantiche normative:** Contracts 1–5 e D2-A v1 FROZEN, con le sole eccezioni esplicite del loading amendment; protocollo deterministico v1 per la sua realizzazione concreta.
3. **Modello numerico e pubblicazione:** D2-B1 v3 → v2 → v1 → v0 nei rispettivi ambiti; D2-B2 publication contract e freeze degli asset. La politica operativa immutabile approvata è registrata al §5.2.
4. **Source Simulation:** contratto originale di estensibilità, review e freeze record; separato dal protocollo Layer 2 successivo.
5. **Evidenze:** report di implementazione/review per la revisione e l’ambito dichiarati. “PASS”, “DRAFT”, HEAD e hash vanno letti nel loro contesto temporale.
6. **Orientamento:** questa guida e i riassunti storici; le nuove istruzioni esplicite dell’utente determinano gli incarichi.

Gli hash identificano byte, non approvazioni o deployment. Il freeze del protocollo/amendment è nel commit `9300b0e9b`; la technical review ne descriveva il candidato precedente e non ne annulla lo stato frozen attuale. Per i documenti più antichi, usare le approvazioni e le precedenze esplicite anziché dedurre lo stato dalle sole intestazioni.

Preparare futuri incarichi con risultato atteso, confini modificabili, input pertinenti e verifiche proporzionate. Non ripetere produzione, qualificazione o audit storico generale per riprendere il lavoro. `.idea/**` è fuori ambito e non va ispezionata. Preservare cache, foto e altri artefatti untracked; non usare vecchi inventari per pulire o ripristinare il checkout. Eventuali percorsi `/tmp` nei report sono evidenze storiche esterne, non nuovi documenti da ricreare.

L’aggiornamento della guida non comporta manutenzione automatica di report, contratti, vettori, asset o manifest. Il suo SHA-256 finale viene comunicato alla consegna, senza incorporarlo nel documento.

## Appendice A — navigazione corrente e identità verificate

### Punti di ingresso dell’implementazione

| Responsabilità | File corrente |
| --- | --- |
| Layer 0 | [identity.go](tools/fakedata/identity.go), [component.go](tools/fakedata/component.go), [stream.go](tools/fakedata/stream.go) |
| Layer 1 e dati italiani | [layer1.go](tools/fakedata/layer1.go), [layer1_data.go](tools/fakedata/layer1_data.go) |
| Acquisizione modello | [socioeconomic_model.go](tools/fakedata/socioeconomic_model.go), [decoder](tools/fakedata/socioeconomic_model_json.go), [compiler](tools/fakedata/socioeconomic_model_compile.go) |
| Runtime Layer 2 | [socioeconomic.go](tools/fakedata/socioeconomic.go), [tipi](tools/fakedata/socioeconomic_types.go), [facts](tools/fakedata/socioeconomic_facts.go), [identity](tools/fakedata/socioeconomic_identity.go), [sampling](tools/fakedata/socioeconomic_sampling.go) |
| Source Simulation / CSV | [source_simulation.go](tools/fakedata/source_simulation.go), [source_csv.go](tools/fakedata/source_csv.go) |
| API e configurazione servizio | [simulated_systems.go](cmd/simulated_systems.go), [server.go](cmd/server.go), [config.go](cmd/config.go) |
| Facade, worker e documento | [simulated_system.go](core/simulated_system.go), [generation](core/simulated_system_generation.go), [document](core/simulated_system_document.go) |
| Simulated Application e patch | [application.go](core/internal/simulated/application.go), [destination](core/internal/simulated/application_destination.go), [Source schema](core/internal/simulated/application_schema_source.go), [Destination schema](core/internal/simulated/application_schema_destination.go) |
| Binding delle connection | [connections.go](core/internal/connections/connections.go), [application.go](core/internal/connections/application.go), [workspace.go](core/workspace.go) |
| Metadata e record persistenti | [schema operativo](core/internal/initdb/schema.sql), [state](core/internal/state/state.go), [datastore](core/internal/datastore/store.go), [scritture](core/internal/datastore/simulated_records_write.go) |
| Admin | [SimulatedWorld.tsx](admin/src/components/routes/SimulatedWorld/SimulatedWorld.tsx), [validazione valori](admin/src/components/routes/SimulatedWorld/simulatedSystemValues.ts), [tipi API](admin/src/lib/api/types/simulatedSystem.ts) |
| Test applicativi socioeconomici | [core](core/simulated_system_socioeconomic_test.go), [schema/patch](core/internal/simulated/application_socioeconomic_test.go), [servizio e round trip](cmd/simulated_systems_socioeconomic_test.go), [generazione/replay](cmd/simulated_systems_generation_test.go) |

I contratti correnti sono collegati nel §5 e le review nel §9. Per la ricetta e pubblicazione: [D2-B1 v3](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3.md), [publication contract D2-B2](SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PUBLICATION_CONTRACT.md), [production report](SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PRODUCTION_REPORT.md). La [External Assets specification](SYNTHETIC_WORLD_EXTERNAL_ASSETS_SPEC.md) conserva il contesto di distribuzione; per conflitti di acquisizione prevale l’amendment frozen, per il comportamento attuale il codice.

### Selezione SHA-256 riverificata l’11 ottobre 2026

Questa piccola selezione identifica le autorità Source Simulation e quelle Layer 2 successive. Non è un inventario del repository o un controllo di disponibilità delle origin.

| File | SHA-256 verificato |
| --- | --- |
| [Source Simulation contract v0](SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0.md) | `9c542ec3c9517d3aaa0d5c108093fb370c212db6a12d3005728f2562071b2d37` |
| [Freeze record Source Simulation](SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0_FREEZE_RECORD.md) | `37d7e2d2341a90669b8548dcd36f7f3585913c066d0067691bc8a1dd612ff74a` |
| [Protocollo deterministico v1 frozen](SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md) | `8abf0347612f75c6f033b6114e9dbe28332994ee0f6f1cce6285790ea03a796f` |
| [Loading amendment v1 frozen](SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md) | `4c91c5398b53af51318bf9a4c5ec8e91084835f5c102084a62df443520e33013` |
| [Vettori deterministici v1](SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_VECTORS_v1.json) | `131b0de96484f61900f27daa3e182b9a526a1d1200f4b2819c084ceed2f2b218` |

## Appendice B — contesto storico di settembre, superato come handoff

La guida del 20 settembre descriveva account simulated legati a S3, backend/Admin non ancora committati e connection ancora da realizzare. Discuteva poi una simulazione HubSpot-first con `SimulatedRecordSchema`, mapper provider e firma di `MarshalValidate`/buffer ancora da scegliere. Quelle proposte sono realmente esistite; non sono l’architettura o le istruzioni correnti. Il percorso attuale è Simulated System / Simulated Application, con schema proprio, Source/Destination implementate e `MarshalValidate` disponibile.

Resta utile la motivazione di separare dati canonici, osservazioni del provider, persistenza e trasporto: un SaaS possiede utenti/contatti, mentre uno storage possiede file con path e formato. Questa distinzione non impone ora HubSpot né una simulazione Storage/File. Non ripristinare il vecchio connector Synthetic, l’intercettazione globale di S3, la modalità workspace `synthetic`, `KRENALIS_SYNTHETIC_CONFIG` o il simulatore HTTP SaaS proposto e abbandonato.

I [report account generation](SIMULATED_ACCOUNTS_GENERATION_IMPLEMENTATION_RESPONSE.md), [Admin review](SIMULATED_ACCOUNTS_ADMIN_REVIEW_RESPONSE.md), [Source Simulation](SYNTHETIC_WORLD_V1_SOURCE_SIMULATION_IMPLEMENTATION_REPORT.md) e [riassunto storico](SYNTHETIC_WORLD_RIASSUNTO_CONTESTO.md) restano intatti come evidenze delle rispettive fasi. Limiti da un milione, stato Ready, vecchie tabelle/account API, assenza di binding e segnalazioni di quelle consegne non vanno trasferiti automaticamente al codice corrente. I riferimenti “Layer 2 da implementare” nei documenti precedenti sono storici.

Le vecchie tabelle SHA-256 del 19/20 settembre sono state sostituite qui dalla navigazione corrente e dalla selezione riverificata dell’Appendice A. Il [manifest originario](SYNTHETIC_WORLD_FILE_MANIFEST.json) e le tabelle nei report conservano inventari storici; non attestano il checkout attuale e non sono stati aggiornati. Non servono un nuovo inventario globale né la ricostruzione di tutti i tentativi D2-B2 per iniziare un incarico.

Non sono requisiti del perimetro corrente lifecycle Layer 3, household, cronologie di eventi/transazioni, cause dei duplicati, orphan/NO_MATCH, correlazioni sofisticate o un motore universale di plugin/schema. La precedente libertà di evolvere dettagli implementativi non autorizza a cambiare semantiche Layer 0–2 frozen o a trasformare nuove possibilità in lavoro obbligatorio.
