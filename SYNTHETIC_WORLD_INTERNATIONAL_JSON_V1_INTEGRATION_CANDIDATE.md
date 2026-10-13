# Synthetic World — Integrazione candidata dei JSON internazionali v1

**Status: CANDIDATE — Not approved / Not frozen.**

## 1. Status, purpose and scope

Questa proposta definisce il confine di integrazione richiesto dalla [specifica
internazionale candidata](SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md)
(di seguito **Spec**, §§F.7 e I.3) e dalla [review indipendente](SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_REVIEW.md)
(§§A, C ed E). Le prescrizioni «deve» e «non deve» descrivono condizioni della
proposta: acquisterebbero autorità soltanto dopo approvazione separata. Il
documento non sostituisce né modifica automaticamente contratti precedenti.

La decisione successiva del proprietario richiede il ritiro operativo della
generazione legacy. La review citata resta immutata come evidenza della
candidatura allora esaminata: non verifica automaticamente questo ritiro.

L'ambito comprende esclusivamente i quattro formati v1 di Spec §§B.2–B.5,
External Assets, la preparazione italiana, un nuovo World base versionato, il
modello socioeconomico italiano esistente applicato a tale World, il nuovo
catalogo fotografico e la policy `generation-v5`, per Simulated System User.
Non definisce altri schemi, algoritmi di generazione o runtime nazionali.

## 2. Normative boundaries and legacy retirement

Layer 1, Contracts 1–5, D2-A, protocollo deterministico, External Model Loading
Amendment e contratto Source Simulation restano intatti e documentano le
semantiche storiche e le regole condivise qui richiamate. L'amendment conserva
la sola precedenza già delimitata nel suo §2. Non ne deriva un obbligo di
conservare operativamente i generatori storici. La nuova baseline disciplina
un percorso successivo, proposto come unico percorso italiano di generazione
non vuota, con le seguenti ammissioni circoscritte:

| Confine storico | Ammissione proposta, limitata al percorso internazionale italiano |
| --- | --- |
| Provenienza del World da `NewWorld`: Contract 1 §7, richiamato dall'amendment §4B | Accettare il World completato dal nuovo ingresso descritto nel §3; nessuna ammissione di Worlds arbitrari o obbligo di mantenere il caricatore CSV. |
| Input Layer 1 di Contract 4 §§5–6, ottenuti dai dati frozen | Ottenere AnchorAge e Region dal medesimo nuovo World autorevole, secondo §3 e Spec §§F.4, G.2–G.3, H.1; conservare significati e parent set. |
| Modello fisso `synthetic-world/it-2026a/model.json`, schema `SW-L2-D2B2-logical-v0`: amendment §§3–4A | Acquisire la rappresentazione internazionale tramite descrittore e proiettarla senza perdita nello stesso modello italiano, secondo §§4–5. |
| Unica identità fotografica dei percorsi precedenti | Separare selezione e contenuto completo soltanto nella nuova baseline, secondo §6 e Spec §F.5. |

Queste ammissioni non autorizzano modifiche a DAG, tassonomie italiane, modello
numerico, limiti L2, primitive, codec degli ID o semantiche delle operazioni
esistenti. La nuova provenienza e il nuovo caricamento sono decisioni di
questo candidato, non autorizzazioni già implicite nei frozen o nei controlli
meccanici del codice corrente.

Il runtime finale deve avere un solo percorso italiano di generazione non
vuota: `generation-v5`, alimentato dai JSON internazionali tramite External
Assets. `empty-v1` resta supportato. Non si mantengono generatori o riprese
v3/v4, né due implementazioni parallele della generazione italiana. Non è
richiesta l'uguaglianza delle persone fra vecchia e nuova baseline.

I sei CSV devono essere convertiti offline senza alterare valori logici e
ordini di Spec §H.1. Non devono essere incorporati nel binario operativo con
`go:embed`, caricati come input runtime o usati come fallback, neppure per
Source Simulation. Gli originali possono restare come provenienza e input di
verifica: la loro rimozione fisica non è decisa qui. Il ritiro riguarda il
caricamento legacy e il dispatch, preservando strutture e algoritmi riutilizzati.
Gli schemi e le formule della Spec restano il riferimento della proposta;
questo documento ne disciplina soltanto l'integrazione, senza modificare
retroattivamente le autorità frozen.

## 3. Versioned World admission

Le versioni sono quelle di Spec §F.7:

| Elemento | Valore | Responsabilità nella nuova baseline |
| --- | --- | --- |
| PersonSpecVersion | `person-v2` | Distingue la nuova specifica della persona nella composizione del WorldID. |
| Protocollo dati di base | `layer1-json-v1` | Identifica domini e proiezioni dei sei fingerprint logici di Spec §F.4. |
| Policy applicativa | `generation-v5` | Seleziona preparazione, costruzione e regole di ripresa nuove. |
| WorldModelVersion della radice | `world-v1` | Conserva la derivazione della radice da versione e WorldSeed. |

L'implementazione deve scegliere questi valori attraverso un percorso
esplicitamente versionato. Il ritiro v3/v4 non autorizza la sostituzione
indiscriminata di `PersonSpecVersion`, `MarketDataVersion` o altre costanti
condivise: occorre conservarne le responsabilità nei consumatori ancora
necessari, in particolare il binding Source del §7.

Il WorldID nuovo conserva composizione e codec esistenti, usando `person-v2` e
la `release` del documento base come componente `market-data-version`, insieme
a namespace, mercato italiano, WorldModelVersion e WorldSeed. Non usa la
release nazionale coordinata né l'identità del pacchetto: la combinazione può
cambiare soltanto per foto o L2. Foto e dati L2 rimangono esclusi dal WorldID
base; una variazione esclusivamente fotografica non deve cambiare l'identità
socioeconomica (Spec §§B.5, F.3, F.7; Contract 1 §§15–17).

I sei fingerprint sostituiscono soltanto i valori dei componenti già indicati
in Spec §F.4. Non richiedono una nuova radice globale: KFD1, percorsi,
indirizzamento per entità/indice e dipendenze separano già gli stream. Layer 2
deve riusare la factory effettiva del World base, senza seed aggiuntivi o digest
di pacchetto nella radice (protocollo §§3, 6). PersonID e CompanyID restano
coordinate reversibili nel namespace e nel rispettivo dominio d'entità; non
diventano hash dei fatti. CompanyID non prova l'appartenenza all'universo
operativo, e gli stessi ID possono avere fatti differenti in baseline diverse.

È ammissibile soltanto un World completato dal nuovo ingresso italiano con:

- dati esterni già decodificati, validati e verificati contro il profilo
  `it-2026a-base-v1`, preparati in strutture immutabili compatibili con
  `italyData`, senza CSV intermedi;
- namespace e coordinate validi, WorldID e SnapshotID correttamente composti,
  factory deterministica inizializzata, ReferenceDate valida e catalogo
  preparato compatibile; nessun World nil, zero-value o parziale;
- ancoraggio `2026-01-01`, dominio italiano 21–75 e copertura delle nascite e
  coorti prescritta; raccordo esatto fra territori base e tassonomia regionale
  del modello, secondo Spec §§C.2–C.3 e G.2–G.3;
- controlli di composizione completati prima della costruzione, salvo quelli
  che dipendono dalle Company realizzate, che restano nella costruzione L2.

Si devono riutilizzare, ove compatibili, `italyData`, slice, mappe, pesi, bucket,
campionatori, primitive, stream e costruzione di indirizzi e contatti. AnchorAge
proviene dalla nascita del nuovo World; Region dal comune scelto per l'indirizzo
di quella persona, copiando il territorio verificato in `placeRow.region`.
Non si estrae un secondo comune per L2 e non si aggiungono altri input geografici
o temporali al DAG. Valgono le sole dipendenze di Contract 4 §§5–6 e 18.

La costruzione riceve dati già preparati e non acquisisce asset. `Person`,
`Facts` e `Company` operano su stato immutabile locale, senza download o
riscoperta. Il caricatore CSV attuale non è un ingresso operativo da preservare
per compatibilità; i suoi consumatori devono ricevere il nuovo World preparato.
La proposta fissa responsabilità e precondizioni, non firme Go definitive o
collocazione dei nuovi tipi privati.

## 4. External Assets acquisition and preparation

La preparazione V5 deve usare esclusivamente le origin ordinate di
`KRENALIS_EXTERNAL_ASSETS_URLS` e il resolver esistente. La policy fissa il
percorso del descrittore; non si introducono variabili d'ambiente, resolver o
selettori liberi di release (Spec §§B.1, B.5, I.1–I.2). Questa è l'unica
acquisizione dei dati base per il runtime italiano, anche per SourceWorld.

La sequenza è: descrittore → documento base, modello socioeconomico e catalogo
fotografico riferiti → validazioni autonome e di profilo → verifiche incrociate
→ stato immutabile preparato → costruzione locale. I digest logici attesi dei
tre documenti, definiti in Spec §F.2, devono essere verificati nel consumo del
candidato di ciascuna origin, prima della sua accettazione definitiva.

Un errore di trasporto, decodifica, validazione o digest rifiuta il candidato e
consente il fallback ordinato. Il successo si pubblica soltanto dopo il ritorno
riuscito di `Resolve`, inclusi i controlli finali del trasporto. Se tutte le
origin falliscono, la preparazione fallisce senza risultati parziali, senza
documenti alternativi, altra release o dati embedded.

Gli owner applicativi dell'acquisizione devono conservare i successi immutabili,
coordinare chiamate concorrenti e consentire retry dopo fallimento. Il risultato
deve appartenere al riferimento completo: almeno ruolo/schema, percorso e digest
atteso, nell'ambito delle origin dell'owner. Digest attesi differenti sullo
stesso percorso non condividono indiscriminatamente il risultato. Si riusa
l'ownership esistente, senza un framework generale di cache.

I documenti possono arrivare da origin diverse se soddisfano i riferimenti.
Due documenti autonomamente validi ma incompatibili impediscono la costruzione
di quella combinazione: non si cerca una sostituzione semantica. Il rifiuto
della combinazione non invalida il modello autonomo già acquisito. Nessun World
o insieme preparato incompleto viene esposto; i successi autonomi possono essere
riutilizzati. Si applicano i controlli di Spec §§G.1–G.3, inclusi ancoraggio,
domini, territori, generi e rendition italiana 256×256.

L'acquisizione deve essere contestuale alla preparazione richiesta, cancellabile
e limitata prima di allocazioni o conversioni. Non parte indiscriminatamente
allo startup. Restano i limiti L2 e la contabilità di risorse dei frozen, con la
separazione delle fasi dell'amendment §§4–6; la nuova rappresentazione non
autorizza incrementi impliciti dei budget.

## 5. Socioeconomic model projection and reuse

Il nuovo decoder separato e versionato deve validare il profilo
`it-2026a-socioeconomic-v1` e trasferire valori tipizzati all'ingresso del
compilatore corrente, senza riserializzare il vecchio JSON per riparsarlo.
Deve rispettare integralmente:

`P(M_internazionale) = L_italiano_originale`

L'uguaglianza riguarda tutti i valori logici, classificazioni, ordinali, unità,
soglie, ancoraggio e dipendenze, secondo Spec §§D, F.6 e H.2. Include le otto
mappe `compiled` ricostruite e la conversione `null → 65535` nei soli slot
previsti. Il riferimento `sourceModel` non sostituisce la prova di equivalenza.
Non si modifica `it-2026a/model.json`, non si ricalibrano masse o leggi e non si
estende silenziosamente il decoder del percorso fisso `logical-v0`. Il file
storico `synthetic-world/it-2026a/model.json` resta frozen, insieme alle evidenze
di provenienza e pubblicazione.

Si devono riutilizzare `SocioeconomicModel`, compilatore, distribuzioni, leggi
discrete, canonicalizzazione L2 e quindici fingerprint frozen. Il modello deve
essere acquisito, validato e compilato prima della costruzione del wrapper.
La nuova acquisizione è una provenienza ammessa solo attraverso
questa proiezione; il tipo Go condiviso non autorizza modelli arbitrari.

La rimozione del percorso applicativo v4 non autorizza una riscrittura di
`SocioeconomicModel`, compilatore, DAG, Company, Employer, Facts, Income,
protocollo numerico o fingerprint frozen. Il vecchio loader può essere rimosso
dal runtime se non serve ad altre funzionalità; non deve essere conservato
esclusivamente per generare sistemi v4. Il ritiro del codice non implica la
cancellazione degli asset frozen o delle evidenze storiche.

Il wrapper conserva lo stesso World base autorevole e il modello immutabile;
non ricostruisce una base da copie della configurazione. Company, indici
Employer, Facts e Income conservano gli algoritmi correnti. Roster e quaranta
indici sono costruiti localmente, con i controlli di completezza, overflow,
risorse e totale positivo prescritti; il fallimento non pubblica un wrapper
parziale e non ritenta con altro seed. Modello e World restano utilizzabili
indipendentemente dalla vita dell'owner di acquisizione.

`M_all` e i quindici fingerprint derivano dal modello logico proiettato;
`D_socioeconomic` identifica invece il documento internazionale. Il
SocioeconomicWorldID continua a legare il WorldID base effettivo, `M_all`,
ancoraggio e `socioeconomic-v1`, secondo protocollo §§4–6. Nessun digest foto,
SnapshotID o identità del pacchetto entra in tale identità o negli stream L2.

## 6. Photo identities and snapshot semantics

La separazione di Spec §F.5 è ammessa soltanto nella nuova baseline:

| Responsabilità | Identità |
| --- | --- |
| Stream di selezione del PhotoID | `F_photo`, con `selection.version` come discriminante di versione |
| Identità del catalogo nello Snapshot | `D_photos` |
| Controllo fotografico del checkpoint V5 (`CatalogSHA256`) | `D_photos` |
| Identità socioeconomica | Nessun digest fotografico |

`F_photo` copre candidati e discriminanti della selezione; `D_photos` copre il
catalogo completo, inclusi metadati di distribuzione. Formule, proiezioni e
ordinamenti sono quelli di Spec §§B.4 e F.1–F.5. Lo stream `photo/select`
conserva filtro per genere, minima distanza dall'età di ancoraggio e scelta
uniforme deterministica fra i pareggi ordinati per ID.

La release precedente `v1` è distinta dal discriminante runtime storico
`fakefacegen-v1`. La copia in `selection.version` di Spec §H.3 sceglie un nuovo
discriminante, senza conservare il precedente componente dello stream (N-01).
Le semantiche dei cataloghi e percorsi fotografici legacy restano riferimenti
storici, senza obbligo di mantenerne i generatori nel runtime.

Cambiare soltanto un percorso WebP lascia invariato il PhotoID canonico a parità
degli altri input di selezione, ma cambia `D_photos` e SnapshotID; il controllo
di ripresa può quindi rifiutare il catalogo. SourceWorld dipende da SnapshotID:
ID e osservazioni Source non hanno la stessa promessa di invarianza del PhotoID.

Il JSON non introduce download o fallback per ciascun WebP. Gli URL continuano
a unire la prima origin configurata al percorso della rendition dichiarata;
la pubblicazione deve assicurarne lì la disponibilità, anche quando i metadati
sono acquisiti da un'altra origin (Spec §I.4). Il catalogo da solo non verifica
i byte delle immagini né rileva una loro sostituzione non dichiarata.

## 7. Simulated System generation-v5 and resume

La creazione ammette soltanto `generation-v5` per un sistema non vuoto e
`empty-v1` per un sistema vuoto. Il dispatch del worker deve distinguere gli
stati persistiti e riconoscere le policy ritirate senza mantenerne i generatori:

| Policy | Percorso e ripresa |
| --- | --- |
| `empty-v1` | Sistema vuoto Available, senza acquisire inutilmente base, modello o fotografie. |
| `generation-v3` / `generation-v4` | Policy ritirate: nessuna nuova preparazione o ripresa; se Preparing, errore permanente verso Failed. |
| `generation-v5` | Nuova preparazione internazionale e nuovo World italiano, con identità della nuova baseline. |

V5 si applica solo ai sistemi creati con tale policy. Non converte o reinterpreta
policy e checkpoint v3/v4, né migra preparazioni storiche. Aggiungere V5 alla
condizione del piano precedente senza selezionare la nuova preparazione e
ritirare i generatori precedenti non soddisfa questo contratto candidato.

I sistemi già Available, anche v3/v4, conservano i record persistiti senza
rigenerazione o cancellazione automatica. Source e Destination devono continuare
a leggere o modificare tali record secondo le capacità esistenti, senza
costruzione del World o acquisizione dei vecchi dataset. Rinomina ed eliminazione
restano disponibili secondo i consueti vincoli di stato e connessione. Failed
non viene riavviato automaticamente e resta gestibile; Deleting prosegue il
cleanup ordinario, indipendente dalla policy. La rimozione dei generatori non
deve eliminare servizi ancora necessari a usare i record persistiti.

Una preparazione v3/v4 ancora Preparing deve essere riconosciuta dal worker
prima di acquisire asset o costruire un piano, e indirizzata al percorso
permanente esistente `fail` / `saveCheckpoint`, con motivazione esplicita di
policy legacy non più supportata. Non deve restare in Preparing per l'assenza
del dispatch, essere ignorata, passare a V5 o essere cancellata automaticamente.
Si conservano policy, checkpoint e conteggio storici, senza continuare la
generazione; l'eventuale pulizia dei record parziali segue la normale eliminazione.

La verifica statica a HEAD conferma la compatibilità di questa direzione:
il worker leader rilegge lo stato persistito e tratta solo Preparing/Deleting;
`saveCheckpoint` persiste l'intento di fallimento, chiude le scritture con il
fence e aggiorna a Failed solo un record ancora Preparing con checkpoint,
conteggio ed errore attesi. Un'interruzione riprende l'intento già persistito;
un errore di database o datastore resta retryable e può ritardare la transizione,
senza riabilitare la generazione legacy. `DeleteSimulatedSystem` consente la
successiva eliminazione da Failed, subordinata ai vincoli di connessione, e
rifiuta quella da Preparing. Non emerge la necessità di una migrazione o di
un nuovo stato; l'implementazione deve conservare autorità, fencing e controlli
concorrenti esistenti.

La preparazione V5 deve acquisire tutti gli input necessari prima della
generazione e costruire il wrapper socioeconomico sullo stesso World base usato
da SourceWorld. Deve mantenere determinismo per indice, batch di massimo 256
record, duplicati, Oracle separato, sei fatti socioeconomici applicativi e
meccanismi correnti di persistenza, conteggio, replay e ripresa. Il retry
dell'acquisizione non introduce un riavvio automatico di sistemi terminali Failed.

Il checkpoint V5 deve mantenere i controlli già ammessi: policy, versioni,
namespace, data di riferimento, URL foto, release/digest fotografico, coordinate
persona/rappresentazione e coerenza con il conteggio persistito. I campi esistenti
`PersonSpecVersion` e `MarketDataVersion` devono riflettere rispettivamente
`person-v2` e `base.release`; `CatalogRelease` identifica la release del nuovo
catalogo e `CatalogSHA256` usa `D_photos`. `WorldModelVersion` resta `world-v1`;
le versioni Source e namespace conservano le responsabilità correnti. Il protocollo
`layer1-json-v1` è selezionato dal percorso V5, senza richiedere un nuovo campo
di checkpoint.

Non si aggiungono ai checkpoint controlli di SocioeconomicWorldID, `M_all`,
digest globale L2, fingerprint socioeconomici o identità del pacchetto usata
come controllo L2 sostitutivo. La riproducibilità tra processi dipende anche
dall'immutabilità operativa delle release e dei riferimenti. Il descrittore
verifica la corrispondenza con i digest dichiarati, ma non protegge dalla
sostituzione coordinata del descrittore e dei suoi asset fra processi
(Spec §I.5; guida §5.2).

Per Source Simulation, la nuova integrazione ammetterebbe l'uso della base
italiana del §3 mantenendo il binding `CountryShare.Version == it-2026a`.
Questo binding non diventa automaticamente `base.release`; SnapshotID distingue
già la nuova baseline. `newSourceWorld` riceve il World dal chiamante e il ramo
IT usa quella stessa base: non carica autonomamente CSV. Deve quindi ricevere
il World esterno preparato, condiviso con il wrapper L2, mantenendo membership,
molteplicità, osservazioni e Oracle. Eventuali ulteriori ammissioni devono essere
esplicite e circoscritte nel relativo ingresso. O-03 non autorizza sostituzioni
globali di `MarketDataVersion` né riscritture di Source Simulation o Oracle.

Gli altri consumatori riscontrati sono il wrapper L2, che legge la base per
nascita, territorio e factory, e test/fixture che costruiscono `NewWorld` o
chiamano il loader direttamente. Devono essere adattati al nuovo ingresso e
alle identità V5, preservando le prove degli algoritmi condivisi e l'equivalenza
offline dei dati. I test che oggi esigono generazione/ripresa v3/v4 devono
verificare il rifiuto permanente delle preparazioni legacy, non imporre il
mantenimento dei generatori. L'uso applicativo dei record Available passa
invece dal datastore e non richiede un World.

## 8. Implementation constraints from independent review

Le osservazioni della review §C restano vincoli implementativi già identificati,
non nuove funzionalità o modifiche ai contratti frozen; O-03 si applica al nuovo
ritiro esplicito secondo §§2–3 e 7, senza attribuirlo alla review storica:

| Osservazione | Vincolo |
| --- | --- |
| O-01 — Canonicalizzazione | C1 deve produrre esattamente i byte di Spec §F.1. Non coincide automaticamente con `tools/json.Canonicalize`: ordine per code point, interi grandi esatti e LF finale devono essere preservati. |
| O-02 — Unicode | Il profilo italiano deve rispettare Unicode 16 di D1-B F.1 e rettifica G.2, senza presumere equivalenza con le tabelle Unicode 17 vendorizzate. La conformità va verificata sul repertorio ammesso, senza rigenerare atomi o normalizzare i display caricati. |
| O-03 — World e versioni | Servono ingresso e ammissione espliciti di World e SourceWorld, con responsabilità di versione distinte; il ritiro v3/v4 segue §§2–3 e 7, senza reinterpretarne i checkpoint. |
| O-04 — Validazione numerica | Prima di conversioni ai tipi Go ristretti e compilazione, il nuovo decoder/profilo deve conservare le precondizioni del decoder corrente: cardinalità, bucket, limiti di leggi, cap/fattori e aritmetica controllata. Il compilatore non sostituisce questi controlli. |
| O-05 — Cache | La cache appartiene agli owner di acquisizione, non al resolver; il successo è associato al riferimento completo e al digest atteso, pubblicato dopo `Resolve`, con retry e coordinamento concorrente. |

Riscontri: `tools/json/functions.go:53`; per le tabelle Unicode,
`vendor/golang.org/x/text/cases/tables17.0.0.go:3` e
`vendor/golang.org/x/text/unicode/norm/tables17.0.0.go:3`; per le precondizioni,
`tools/fakedata/socioeconomic_model_json.go:277` e
`tools/fakedata/socioeconomic_model_compile.go:37`. Non è richiesta una nuova
architettura per soddisfare questi vincoli.

## 9. Acceptance conditions and references

Dopo approvazione, l'implementazione dovrà dimostrare i criteri di Spec §J e
della review §F: equivalenza integrale dei dati base e della proiezione L2,
canonicalizzazione e quindici fingerprint L2 invariati, byte C1 conformi,
acquisizione senza pubblicazione parziale, isolamento delle identità e ripresa
deterministica V5. I criteri della review relativi alla conservazione operativa
v3/v4 sono superati dalla decisione qui recepita; la review resta evidenza storica.
Devono risultare assenti CSV embedded, fallback CSV e generatori/riprese v3/v4;
V5 deve essere l'unica nuova preparazione non vuota, con `empty-v1` conservato.
Occorre verificare Preparing legacy verso Failed con motivazione esplicita,
conservazione e uso dei record Available, gestione Failed/Deleting e binding
Source sul nuovo World, inclusi gli adattamenti dei test legacy. Restano assenti
nuovi controlli L2 nei checkpoint e rete nella generazione. Sono condizioni
future: questo consolidamento documentale non certifica asset convertiti,
esecuzione, deployment o freeze.

I riferimenti contrattuali puntuali usati sopra sono:

- [D1-B](L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md), §F, e
  [rettifica](L1-D1-B_Contract_Rectification.md), §G.2: testi, atomi e unicità.
- [Contract 1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md), §§6–7, 11–17;
  [Contract 2](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md), §§6–21;
  [Contract 3](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md), §§5–16;
  [Contract 4](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md), §§5–6, 18;
  [Contract 5](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md), §§3–5, 7–14.
- [D2-A](SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md), §§19–21;
  [protocollo deterministico](SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md),
  §§3–6, 8–13; [loading amendment](SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md),
  §§2–7: numeri, identità, costruzione, acquisizione e risorse.
- [Source Simulation](SYNTHETIC_WORLD_V1_EXTENSIBILITY_SOURCE_CONTRACT_v0.md),
  «Scope, terms, and country boundary», «Determinism, record identity, and
  Oracle», «Record generation and transport».

Comportamento corrente riscontrato a HEAD
`50ed99476cf22a8137d1adcf8e2c82f7235e0ec6`: `tools/fakedata/layer1.go:154`,
`layer1_data.go:55`, `socioeconomic.go:28`, `socioeconomic_model.go:67`,
`external_photo.go:49,203`, `layer1_photo.go:287`, `source_simulation.go:59`
(questi ultimi nello stesso package); `tools/externalassets/resolver.go:42`,
`core/simulated_system.go:70`, `core/simulated_system_generation.go:434,453` e
`cmd/server.go:61`. Il codice attuale è evidenza del punto di partenza, non
approvazione del nuovo ingresso.

Dipendenze del ritiro riscontrate nello stesso HEAD:
`tools/fakedata/layer1_data.go:39,41,210,237` (caricamento CSV) e
`layer1_data.go:55,100` (strutture e bucket riutilizzabili);
`core/simulated_system.go:25,70,138,264` (policy, creazione, eliminazione e
rinomina), `core/simulated_system_generation.go:85,161,177,255,336,453`
(selezione, fallimento, preparazione, rilettura e transizione);
`core/internal/simulated/application.go:85,110,124` e
`core/internal/datastore/simulated_records_write.go:30,86` (record persistiti).
Fra i consumatori e le fixture da adattare: `tools/fakedata/source_simulation.go:59,84,176`,
`tools/fakedata/socioeconomic_facts.go:55,62`, `tools/fakedata/layer1_test.go:558`,
`tools/fakedata/layer1_data_test.go:103,132`,
`tools/fakedata/socioeconomic_model_test.go:66`,
`core/simulated_system_generation_test.go:387`,
`core/simulated_system_socioeconomic_test.go:27`,
`cmd/simulated_systems_generation_test.go:160`,
`cmd/simulated_systems_socioeconomic_test.go:111,457` e
`test/synthetic_identity_resolution_test.go:43`. Sono dipendenze per
l'implementazione successiva, non modifiche eseguite in questa candidatura.

Documentazione di supporto, senza precedenza sui contratti:
[guida](SYNTHETIC_WORLD_GUIDA_PROGETTO.md), §§5.1–5.2 e 8.3;
[ricognizione](SYNTHETIC_WORLD_EXTERNAL_ASSETS_RICOGNIZIONE.md), §C.1 e C.3;
[dossier italiano](SYNTHETIC_WORLD_MODELLO_ITALIANO_DOSSIER.md), §§G.3–G.5.
