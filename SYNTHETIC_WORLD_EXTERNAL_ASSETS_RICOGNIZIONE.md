# Synthetic World: ricognizione tecnica su External Assets e JSON multinazionali

Ricognizione in sola lettura sul branch `synthetic-world`, HEAD
`50ed99476cf22a8137d1adcf8e2c82f7235e0ec6`. Sono stati esaminati codice,
test, contratti e documentazione locale; non sono stati eseguiti test, build
né generatori. L’unica scrittura sul repository è questo rapporto richiesto.
I riferimenti indicano il codice corrente, salvo quando esplicitamente
qualificati come contratto, guida o report storico.

## A. Sintesi esecutiva

**Il resolver e i proprietari di acquisizione per il modello socioeconomico e
per il catalogo fotografico sono già presenti. I sei dataset di base italiani
sono invece ancora incorporati nel binario come CSV e caricati da una funzione
locale.** Il motore di generazione consuma strutture in memoria ben definite;
un nuovo formato di trasporto può quindi ricostruire quelle stesse strutture
prima della costruzione del World, senza intervenire sui campionatori.

La strada che richiede meno cambiamenti è riutilizzare la configurazione
`KRENALIS_EXTERNAL_ASSETS_URLS`, il resolver e il modello di ownership/cache
già usati dai due source esistenti. Per i dati di base, occorre aggiungere
acquisizione e validazione JSON che restituiscano l’equivalente logico di
`italyData`, e collegarle solo al percorso di preparazione che genera persone.
Le regole di generazione, i percorsi casuali e le semantiche congelate devono
restare invariati. La decisione più delicata è come mantenere i sei identificatori
di dipendenza SHA-256 attuali quando i byte trasportati diventano JSON: oggi gli
hash frozen identificano i byte dei CSV, e i rispettivi digest partecipano agli
stream deterministici.

| Già pronto nel codice | Manca o va verificato |
| --- | --- |
| Resolver con origini ordinate e fallback completo su trasporto e consumo; source immutabili con cache di successo e retry dopo fallimento. | Acquisizione esterna, decodifica e preparazione dei dati di base italiani; al momento i CSV sono `go:embed`. |
| Acquisizione JSON del modello `it-2026a`, compilazione numerica locale e acquisizione del catalogo fotografico esterno. | Il catalogo fotografico esterno descrive metadati e percorso dei WebP, ma non scarica né verifica i byte delle immagini. Il loro URL usa la prima origine. |
| Il motore Go genera persone e fatti localmente dopo l’acquisizione. | Disponibilità reale del modello, del catalogo e dei WebP sulle origin di deployment non verificata. |
| Confine architetturale frozen per mantenere generazione nazionale e motore delle sorgenti/Oracle separati. | Confine concettuale non equivale a un loader multinazionale già implementato: il runtime attuale è italiano e alcuni assi socioeconomici sono fissi. |

La guida dell’11 ottobre conferma il confine tra acquisizione, costruzione e
generazione, ma si dichiara non canonica; prevalgono il comportamento nel codice
e i contratti frozen (`SYNTHETIC_WORLD_GUIDA_PROGETTO.md`:5–13, 127–147).

## B. Inventario dei dati

La tabella distingue input runtime da materiale di provenienza, parametri e
fixture. Le dipendenze deterministiche descritte sono quelle verificate nel
codice, non nuove identità proposte.

| Input effettivo | Origine, formato e produzione offline identificabile | Caricamento e consumatore Go | Dipendenze deterministiche |
| --- | --- | --- | --- |
| Pesi delle età | `tools/fakedata/data/it-2026a/age_weights.csv`: CSV `age,weight`; POSAS ISTAT al 2026-01-01. Il dataset finale è in Git; gli archivi sorgente non lo sono. La documentazione di provenienza non identifica un builder eseguibile riproducibile. | Incorporato da `layer1_data.go`, hash grezzo verificato, poi 55 righe nella slice `[]ageRow` e pesi paralleli `[]uint64`. Consumato da `birth`. | Righe posizionali per età 21–75; pesi positivi interi. Ordine e valore del peso determinano `WeightedIndex`; il componente SHA “age-data” entra nello stream dell’età. |
| Nomi propri | `first_names.csv`: `gender,cohort,name,ascii_atom,weight`. 800 righe, dieci gruppi (due generi × cinque coorti), 80 righe per gruppo e quota 20/40/20 per peso 4/2/1. Evidenza ISTAT più curation; il builder non è presente. | Parsing tipizzato in mappe di righe/pesi indicizzate per genere e coorte. `firstName` sceglie una riga; il valore display e l’atomo ASCII alimentano nome ed email. | L’ordine delle righe dentro ciascun gruppo è conservato; la scelta pesata usa l’ordine dei pesi e lo SHA “first-name”. Genere/coorti e i loro valori sono regole Go. |
| Cognomi | `last_names.csv`: `name,ascii_atom,weight`; 600 righe, quote 240/240/120 per pesi 1/2/4. Curation originale approvata, non una distribuzione nazionale importata; builder non presente. | Slice `[]nameRow` e pesi paralleli. `lastName` campiona per indice; l’atomo ASCII viene usato nell’email. | Ordine file conservato e significativo per `WeightedIndex`; dipendenza SHA “last-name”. Display Unicode e atomo ASCII sono due valori distinti. |
| Località e geografia | `places.csv`: `istat_code,city,state_prov,region,population`; 7.896 righe. Popolazione da POSAS, universo territoriale e codici da SITUAS, con crosswalk ISTAT delle 377 ricodifiche comunali sarde. | `[]placeRow`, con CAP validati separatamente. La selezione uniforme usa l’indice della slice; quella pesata usa bucket contigui da 128 località, totali e pesi di popolazione. | Codice ISTAT stringa ASCII a sei cifre e ordinamento strettamente crescente; non convertirlo in numero. L’ordinamento determina gli indici e i bucket. Popolazioni sono interi positivi; SHA “places” è input dei due stream geografici. |
| Codici postali | `postal_codes.csv`: `istat_code,postal_codes`; una riga per ciascuna località, CAP separati da punto e virgola. GeoNames fornisce osservazioni postali; 147 override revisionati si basano su IPA Enti. Le fonti precisano che il risultato non è una copertura postale completa e autoritativa. | Mappa `map[string][]string` per codice ISTAT. Verifica copertura uno-a-uno con le località, CAP di cinque cifre ASCII, ordinati e senza duplicati; il CAP è scelto uniformemente. | Zeri iniziali sono parte della stringa; ordinamento CAP conservato e significativo per la scelta indicizzata. SHA “postal” entra nello stream CAP. Gli override sotto `provenance/` sono evidenza/build-time, non un settimo dataset runtime. |
| Frammenti stradali | `street_names.csv`: `name,weight`; 400 righe, quote 160/160/80 per pesi 1/2/4. Lessico originale curato; la verifica umana delle combinazioni è documentata, ma il builder non è presente. | Slice `[]nameRow` e pesi paralleli; la colonna ASCII è assente/vuota. `address` seleziona il frammento con campionamento pesato. | L’ordine delle righe e i pesi sono significativi; SHA “street-name” entra nello stream. Tipi stradali e suffissi sono invece parametri Go (vedi sotto). |
| Modello socioeconomico italiano | Asset JSON esterno, percorso runtime corrente `synthetic-world/it-2026a/model.json`, schema chiuso `SW-L2-D2B2-logical-v0`. Contiene età, pesi categoriali, leggi degli addetti, fattori Employer, mappe e leggi Income. Report D2-B2 documenta una produzione canonica, ma il producer locale ispezionato si autoidentifica come qualificazione non canonica: la riproducibilità del builder va chiarita (sezione C). | `SocioeconomicModelSource.Load` usa il resolver; decoder JSON tipizzato e compilatore ricostruiscono il modello immutabile. `SocioeconomicWorld`, `Facts`, Company, Employer e Income lo consumano. | L’identità deriva dai valori logici canonici, non dai byte/ordine/whitespace del JSON. Interi esatti, vettori posizionali e ordini/ordinali frozen determinano lookup e stream. |
| Catalogo e metadati fotografici | Snapshot locale fakefacegen: `catalog.json`, `specs.json`, `manifest.json`; cinque rendition WebP per record: 64, 128, 256, 512 e 1024. Acquisizione/provenienza della fixture è documentata; non è stato trovato un builder di pubblicazione offline nel codice esaminato. Il catalogo esterno di runtime ha invece schema compatto: release, selectionSeed e record ID/genere/età richiesta. | Loader locale valida lo snapshot completo; source esterno decodifica e ordina il catalogo metadata-only. `selectPhoto` sceglie i volti generati non esclusi dello stesso genere con età richiesta più vicina. | ID, genere, età, release e seed di selezione governano compatibilità e selezione. L’ordine del catalogo esterno viene normalizzato per ID; a parità di ID/attributi/seed la selezione può mantenere gli input Layer 1 frozen. |
| Immagini WebP | File separati dal JSON. Snapshot locale contiene tutte le rendition previste; il formato esterno costruisce un percorso relativo per la rendition 256, senza SHA o byte length nel catalogo esterno. | Il loader locale legge, decodifica e verifica dimensione/digest. Nel flusso esterno il Core compone un URL immagine dal base URL e dal percorso; non passa i byte WebP al resolver. | Il PhotoID è locale al catalogo, non identità della persona. Checkpoint conserva checksum/release del catalogo e base URL; l’URL immagine materializzato incorpora il PhotoID. La selezione usa il seed frozen. |
| Tabelle e regole codificate | Tabelle Go per pattern email, domini .test, tipi di strada e suffissi; ancoraggio nascita e finestra reference-date; coorti, rapporti uniformi/popolazione e struttura degli indirizzi. Non sono dataset separati né produzione offline. | Tabelle/constanti in `layer1.go` e `layer1_helpers.go`; consumate direttamente dagli algoritmi. | Cambiarle cambierebbe le semantiche del generatore anche a dataset invariati. Vanno mantenute nel motore Go per questa ricognizione; non sono candidati automatici a External Assets. |
| Fixture e prove offline | `tools/fakedata/testdata/fakefacegen-v1-selected/` contiene metadati verbatim e immagini di due ID soltanto; `testdata/external-photos-v1.json` è una fixture catalogo di quattro ID. `core/testdata/socioeconomic-model.json` è una fixture artificiale. I JSON D2-B2, crosswalk, publication, producer e report locali sono evidenza di produzione, non input usati per generare ogni persona. | Usati da test o qualificazioni, non dai source runtime normali. I CSV di provenienza e gli override non sono coperti dall’embed diretto dei sei CSV. | Non attribuire a fixture parziali o report locali l’identità di un catalogo/modello di produzione. Non sono stati eseguiti. |

**Dati di provenienza e produttori.** `tools/fakedata/data/it-2026a/SOURCES.md`
identifica POSAS, SITUAS, crosswalk sarda, GeoNames, IPA e il carattere curato
dei dati D1-B (righe 1–62). Dichiara che gli archivi grezzi non sono in Git e
che le osservazioni CAP non sono un inventario nazionale completo. Il contratto
D1-B locale dichiara esplicitamente di non contenere i CSV completi e di non
implementare il builder (`L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md`:9–19).
I test controllano inoltre i 147 override e la località di Reggio Calabria
(`tools/fakedata/italy_data_test.go`:157–209). Non ho trovato un produttore
eseguibile locale per ricostruire offline tutti i sei CSV; non è quindi
possibile descrivere oltre ciò che la provenienza documenta.

## C. Analisi per categoria

### 1. Dati di base delle persone

**Comportamento verificato nel codice.** `layer1_data.go` incorpora tutti i
CSV con `go:embed`, chiama `parseLayer1ItalyData` tramite
`sync.OnceValues` e associa sei digest hard-coded ai componenti di dipendenza
(righe 21–42, 55–74). Il loader verifica prima l’hash dei byte CSV, vieta BOM,
CR e assenza del LF finale, poi usa il parser CSV standard e confronta
intestazioni esatte (210–234, 237–265). Un successo o un errore di parsing
restano memoizzati per la vita del processo: questo loader non ha rete,
fallback o retry.

Le validazioni sono specifiche del contenuto: 55 età posizionali 21–75 e pesi
interi positivi (133–160); 800 nomi propri, dieci gruppi e quote fisse
(163–207); quote 1/2/4 per cognomi e frammenti (331–367); 7.896 località con
codici ISTAT a sei cifre, stringhe geografiche ammesse, popolazioni positive e
ordine crescente (369–404); una lista CAP completa, ordinata e senza duplicati
per ogni località (406–442). I valori display devono essere UTF-8 validi e
senza spazi esterni, ma non vengono normalizzati Unicode o case-folded
(477–479). Gli atomi ASCII sono un campo distinto con validatore limitato a
lettere ASCII minuscole e trattini (456–465).

Il parser conserva l’ordine d’ingresso per righe di nomi, cognomi e strade;
nei nomi propri conserva l’ordine relativo delle righe di ogni gruppo.
Località e CAP devono già rispettare il loro ordinamento contrattuale. I pesi
sono convertiti da decimali interi in `uint64`; non vanno rappresentati come
float o riformattati con arrotondamenti. I codici ISTAT e CAP restano stringhe,
quindi mantengono gli zeri iniziali. Queste proprietà sono parte della
ricostruzione delle slice/mappe, non dettagli di presentazione.

**Dipendenze dal generatore e dai contratti.** `italyData` contiene righe,
pesi paralleli, bucket popolazione e sei componenti SHA (55–74). La generazione
usa indici pesati su tali slice: nascita, nomi, cognomi, CAP e frammento stradale
(`layer1.go`:265–294, 359–385, 417–466); la località pesata è scelta prima
tra bucket contigui e poi nella slice del bucket (469–497). Riordinare righe
cambia perciò i risultati anche lasciando invariati i valori matematici e le
somme totali.

I sei SHA in `layer1_data.go`:29–34 sono SHA dei file CSV esatti, inclusi
rappresentazione, ordine, intestazioni e terminatori; sono trasformati in
componenti bytes (118–131, 293–328). Non sono hash di una serializzazione
logica canonica. I test congelano nomi e byte grezzi di tali componenti e ne
verificano l’isolamento per famiglie di campi
(`layer1_data_test.go`:15–128). Cambiare il digest passato allo stream, anche
perché il nuovo documento JSON ha byte diversi, può cambiare le persone.

Invece `WorldID` include namespace, mercato, versione dati, versione spec,
versione motore e seed, non i sei hash CSV; `SnapshotID` aggiunge reference
date e identità catalogo foto (`layer1.go`:168–211; test
`layer1_test.go`:433–475). Gli ID persona restano derivati da namespace e
indice. I contratti deterministic frozen proteggono stream e output: non vanno
aggiornati per accomodare il formato.

**Costanti che restano nel motore.** Il codice definisce pattern/ponderazioni
email e domini .test, tipi e suffissi stradali (`layer1.go`:62–83), scelta
uniforme del genere, miscela 95/5 tra località pesata/uniforme, due estrazioni
per il numero civico, probabilità suffisso 7/100 e ancoraggio nascita
2026-01-01 (265–385, 347–357). Coorti e regole di reference-date sono in
`layer1_helpers.go`:39–53, 99–112. Sono regole algoritmiche e semantiche
nazionali, non record di dati da esternalizzare. In particolare lo street type
è una tabella del motore; il lessico dei frammenti è un dataset.

**Valutazione JSON.** Un decoder JSON futuro dovrà applicare le stesse regole e
costruire le medesime slice, mappe, pesi e bucket che oggi produce il parser
CSV. Il trasporto può cambiare; i valori logici, i tipi interi, gli ordinamenti,
gli atomi ASCII già curati e le dipendenze stream devono restare. Per
`it-2026a`, la raccomandazione è mantenere i sei componenti di compatibilità
attuali separati dai byte del nuovo trasporto. È una raccomandazione di
compatibilità, non una scelta di nuovo schema.

### 2. Modello socioeconomico

**Formato e acquisizione attuali.** Il decoder in
`socioeconomic_model_json.go` ammette il documento chiuso
`SW-L2-D2B2-logical-v0`, con schema, conteggio aziende, otto mappe d’età,
pesi per assi categoriali, distribuzioni addetti, fattori Employer e sei
rami Income (275–394). Gli array sono vettori posizionali secondo assi frozen,
non record etichettati; i campi extra o valori non previsti non diventano
estensioni implicite. I numeri sono interi unsigned esatti: il decoder usa
`ParseUint` e rifiuta segni meno, frazioni ed esponenti (162–181).

`SocioeconomicModelSource` acquisisce attraverso l’esistente resolver il
percorso fisso italiano, mantiene un solo modello immutabile riuscito e permette
retry dopo errore; costruzione del world e generazione non fanno HTTP
(`socioeconomic_model.go`:17–67, 73–114). Il decoder valida limiti e cardinalità;
`compileSocioeconomicModel` verifica supporti crescenti, pesi positivi,
dimensioni complete, fattori e riferimenti delle mappe, quindi produce pesi
cumulativi interi e stato immutabile
(`socioeconomic_model_compile.go`:12–40, 103–223). Il protocollo fissa
l’identità sui valori logici canonicalizzati, non su whitespace, ordine delle
chiavi, compressione o origine HTTP (`socioeconomic_identity.go`:58–99,
129–163; protocollo §4, righe 60–99).

**Parti riutilizzabili e parti italiane.** La rappresentazione chiusa a interi,
array tipizzati, supporti/masse, mappe e leggi è una forma trasportabile e il
meccanismo decoder → validazione → compilazione è già riusabile per l’asset
italiano. Non lo è automaticamente il significato degli indici. Il runtime usa
assi fissi di 20 regioni italiane, 14 industrie, 10 famiglie professionali,
7 livelli d’istruzione, 6 attività, 4 livelli professionali e 5 fasce
dimensionali; le regioni sono stringhe/ordinali hard-coded e il loro indice
partecipa alle formule row-major (`socioeconomic_facts.go`:12–45, 66–90,
96–160; compiler 135–152). I redditi sono interi EUR annuali lordi
(`socioeconomic_types.go`:7–9; `socioeconomic_model.go`:28–42). Lo schema
non codifica valuta, paese o semantica tassonomica autonoma.

Di conseguenza, una futura release potrebbe riutilizzare la struttura solo
quando cardinalità, ordine e significato degli assi siano compatibili. Con
assi regionali nazionali differenti, tassonomie diverse o redditi in un’altra
valuta, la forma attuale non offre un adattamento semantico: sostituire i valori
italiani con altri numeri o “rimappare” gli ordinali cambierebbe il significato
dei vettori. Questo impedisce il riuso diretto universale, ma non dimostra la
necessità di modificare il modello italiano frozen. Non progetto qui un modello
USA né propongo un nuovo formato numerico.

**Produzione offline e disponibilità.** Il report
`SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PRODUCTION_REPORT.md` classifica il prodotto
come produzione canonica e registra hash, input e verifiche (righe 1–25).
L’artefatto producer locale `D2B2-producer-v3.py`, pur avendo l’hash indicato
nel report, si descrive come producer di qualificazione “NONCANONICAL” e il
CLI esegue il ramo numerico di qualificazione (righe 1–7, 2352–2377). È una
discrepanza documentale concreta: non consente di affermare che quel file
riproduca il modello canonico senza un chiarimento. Il modello logico locale e
le evidenze di produzione non vanno confusi con input runtime; non ho trovato
il file di deployment `model.json` nel repository né verificato la sua presenza
sulle origin.

Il modello `it-2026a` e i contratti del protocollo/loading restano invariati.
L’amendment frozen attribuisce la rete al solo owner External Assets, richiede
schema chiuso, modello immutabile e generazione locale, senza modello embedded
(`SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md`:116–197).

### 3. Asset fotografici

Il percorso locale fakefacegen descrive metadati catalogo (versione, seed,
conteggio, modello e versione delle specifiche), specifiche per ID con età,
sesso e attributi di generazione, e manifest con stato, percorso, WebP,
dimensioni, byte e SHA (`fakefacegen.go`:66–120). Il loader completo richiede
le rendition nelle cinque dimensioni e valida contenuto WebP, dimensioni,
dimensione massima e SHA (`layer1_photo.go`:73–85, 169–198;
`fakefacegen.go`:181–239). Ordina i volti per ID e calcola un checksum
semantico del catalogo prima di creare le dipendenze Layer 1
(`layer1_photo.go`:137–158).

Il source esterno già acquisisce un JSON ridotto di catalogo: formato, release,
selectionSeed e record ID/genere/età richiesta
(`external_photo.go`:20–27, 99–113). Rifiuta record malformati o ID
duplicati, ordina i record per ID e crea metadati per la rendition
256 (`external_photo.go`:186–248). Il seed è esplicitamente l’input RNG Layer 1
esistente, non l’hash dei byte scaricati; la migrazione del catalogo locale deve
conservarlo (`external_photo.go`:201–213). Questo consente di preservare
l’input di selezione esistente se ID, genere, età, release/seed sono equivalenti.
`selectPhoto` sceglie per genere e distanza minima dall’età della persona e
usa lo stream di selezione solo per sciogliere i pareggi
(`layer1_photo.go`:287–337).

**Limite attuale delle immagini esterne.** Il JSON esterno non contiene byte,
SHA né lunghezza WebP; la prova unitaria conferma che il catalogo ha soli
metadati (`external_photo_test.go`:25–49). Il percorso relativo è
`synthetic-world/photos/v1/256/{id}.webp` (codice
`external_photo.go`:231–234). Il documento generato diventa un URL composto
dal base URL e dal percorso immagine (`core/simulated_system_generation.go`:591–600).
In modalità esterna il base URL è la prima origine configurata
(`cmd/server.go`:60–67); il resolver con fallback è usato per il catalogo,
non per ciascun WebP. Pertanto la pubblicazione dei file sul percorso atteso
della prima origine rende le immagini raggiungibili; se si richiede fallback
per immagine fra origin, tale comportamento oggi non c’è. La necessità di un
cambiamento applicativo dipende quindi dal requisito operativo sulle origin e
dal fatto che siano repliche equivalenti; non si giustifica cambiare la
selezione fotografica.

La modalità locale `KRENALIS_SYNTHETIC_PHOTOS_DIR` carica catalogo e immagini
locali e serve `/photos/` (`cmd/server.go`:49–67, 187–192). La fixture
selezionata di test contiene i JSON globali verbatim ma immagini soltanto per
due ID e dichiara di non essere un catalogo completo
(`tools/fakedata/testdata/fakefacegen-v1-selected/FIXTURE.md`:1–41).
La fixture del catalogo esterno ha quattro ID e non rappresenta asset di
produzione (`external_photo_test.go`:25–49).

PhotoID compare nelle persone e nei documenti come riferimento all’immagine.
Il checkpoint salva checksum/release catalogo, URL base, versioni, data di
riferimento e coordinate di ripresa; il documento materializzato contiene
l’URL che incorpora il PhotoID. La ripresa rifiuta release/checksum o input di
generazione diversi (`core/simulated_system_generation.go`:37–49,
434–450, 453–483). Il contratto e il codice non aggiungono un’identità del
modello socioeconomico al checkpoint; il protocollo esclude foto e reference
date dall’identità socioeconomica (§5, righe 103–125). Questa ricognizione non
propone di cambiarli.

## D. Punti di modifica minimi

### Probabilmente necessari

1. **Owner dei dati di base e conversione JSON → strutture esistenti.** Il
   confine minimo è nel package `tools/fakedata`: oggi il parsing embedded
   termina in `italyData` e `NewWorld` chiama quel loader
   (`layer1_data.go`:237–328; `layer1.go`:148–160). La futura acquisizione
   dovrebbe produrre la stessa forma in memoria, con validazione equivalente,
   prima di costruire il World.
2. **Wiring nel percorso applicativo di preparazione.** Le origin sono già
   passate a Core per modelli e foto (`cmd/server.go`:60–90); il worker
   carica foto e, per v4, modello in `prepare`, poi costruisce il piano
   (`core/simulated_system_generation.go`:177–203, 514–532). Il nuovo
   caricamento base va aggiunto a questo percorso o al suo confine di
   costruzione, non allo startup né agli algoritmi casuali.
3. **Prove d’equivalenza.** Vedi sezione F; occorre aggiungere copertura
   specifica del nuovo decoder/fallback e confronto contro le attuali
   strutture/output, senza eseguire ora i test.

### Condizionali o già coperti

- **Catalogo fotografico:** acquisizione esterna e source owner ci sono. Se
  basta l’URL pubblico sulla prima origine, serve la corretta pubblicazione di
  catalogo e WebP, non un nuovo sistema di generazione. Se è requisito il
  fallback WebP, il percorso attuale non lo implementa e va deciso come
  fornirlo.
- **Modello socioeconomico:** riutilizzare l’implementazione esistente; non
  cambiare contenuto logico, schema frozen, compiler o identità del modello.
- **Cache, fallback, retry e origine configurata:** riusare resolver, env e
  ownership esistenti; non serve una seconda variabile né un secondo resolver.

### Da lasciare invariato / non giustificato

Restano invariati i campionatori Layer 1, gli stream frozen, Person/ID,
semantiche del modello socioeconomico, contratti frozen, costruzione dei fatti
e comportamento di resume. Non aggiungere controlli del modello nei checkpoint.
Non aggiungere ora Companies/Employer ai documenti applicativi. Pattern email,
regole di indirizzo, coorti, ancoraggi e rapporti sono regole del generatore;
spostarli in asset non è necessario a soddisfare il requisito sui dataset.
Non sono giustificati framework plugin, interfacce universali o riscrittura
della Simulated Application.

## E. Compatibilità JSON multinazionale

Il requisito di **strutture JSON riutilizzabili** è raggiungibile per i dati di
base senza imporre dati uguali: età/pesi interi, record di nomi con valore
display e atomo per la ricerca, record geografici con codici stringa e
popolazione intera, mapping località-codici postali e frammenti stradali
possono avere forme comuni. I valori, set di codici, cardinalità, fonti,
validazioni nazionali e le regole che interpretano gli assi restano
specifici del paese. Catalogo foto con ID, genere ed età richiesta è già una
forma esterna relativamente neutrale; file WebP restano asset binari.

Il decoder e l’owner del modello socioeconomico sono riusabili come meccanismi
di acquisizione/validazione, ma il documento italiano attuale non è una
struttura semanticamente neutrale: il runtime presume ordini e cardinalità
fissi, nomi delle venti regioni italiane, classificazioni specifiche e valuta
EUR. Uno schema identico con significati diversi negli stessi slot sarebbe
corrotto semanticamente pur essendo valido strutturalmente. Una nazione con
assi regionali diversi o valuta diversa non può essere rappresentata
correttamente dal modello corrente senza una scelta esplicita di compatibilità
o una revisione futura. Non c’è necessità dimostrata di toccare ora
`it-2026a`; non viene definito qui un modello per un’altra nazione.

Il confine architetturale per estensione nazionale è già formalizzato dal
contratto Source Simulation frozen e dal freeze record. Il contratto mantiene
la generazione specifica per nazione locale al generatore del paese e separa
SourceRecord/Oracle (righe 23–48, 62–70); non specifica però API Go, formato
asset o loader. È sufficiente come confine concettuale per riusare il motore
delle sorgenti, non prova che il caricamento base sia già astratto. Il Core
corrente seleziona IT esplicitamente (`core/simulated_system_generation.go`:528–532).

Per minimizzare modifiche, l’opzione più coerente con il codice è mantenere un
solo percorso di acquisizione/preparazione coerente dei dati di base del paese,
che componga le sei componenti in `italyData`; i formati logici delle singole
categorie possono restare condivisibili. Separare completamente le sei
acquisizioni aumenta i punti di fallback e il rischio di combinare release
diverse. La scelta tra confezionamento aggregato e documenti fisicamente
separati è ancora da approvare: non cambia gli algoritmi e questa ricognizione
non sceglie nomi, percorsi o nuovi schemi.

## F. Piano di verifica dell’equivalenza

La futura acquisizione dovrà dimostrare che a parità di valori logici e
configurazione restituisce i risultati italiani attuali:

1. **Input e rappresentazione:** per ciascun CSV, confronto dei valori tipizzati
   e dell’ordine con la forma JSON decodificata; confrontare conteggi,
   stringhe UTF-8 byte per byte, atomi ASCII, zeri iniziali, pesi interi,
   somme e bucket. Includere casi malformati, duplicati, cardinalità errate e
   valori fuori formato.
2. **Identità deterministica di Layer 1:** verificare che i sei componenti
   abbiano gli stessi nomi e byte SHA attuali e che l’isolamento dei campi
   resti quello fissato da `TestLayer1DatasetDependencyIsolation` e
   `TestLayer1DependencyComponents` (`layer1_data_test.go`:15–128).
   Proteggere percorsi stream con `TestLayer1StreamPaths` e risultati
   completi di `TestLayer1PersonGolden` per indici, inclusi estremi e
   PhotoID (`layer1_test.go`:360–429). Usare anche test già presenti di
   determinismo/accesso casuale, nomi e data ancorata
   (`layer1_test.go`:48–63, 123–208, 283–358).
3. **Foto:** usare `TestExternalPhotoCatalogCompatibility` per verificare
   che seed/catalogo esterno preservi gli input di selezione locale, insieme
   ai test di selezione e ai goldens. Verificare URL generati e checkpoint
   per stessa release, checksum, base URL, PhotoID, ripresa e URL documento.
   Un eventuale requisito di fallback dei WebP necessita una prova distinta:
   i test esistenti coprono fallback catalogo, non download delle immagini.
4. **Modello e fatti:** non modificare `model.json`; riutilizzare i test di
   decoder/source, identità, compiler e vettori
   (`socioeconomic_model_test.go`, `socioeconomic_protocol_test.go`,
   `socioeconomic_lifecycle_test.go`). Con lo stesso modello compilato e
   mondo base, confrontare Facts per indici fissi, tutte le sei branche,
   Company/Employer, Income e le identità/fingerprint. I test di Company,
   Facts, isolamento e non valutazione delle branche esistono in
   `socioeconomic_company_test.go`, `socioeconomic_facts_test.go`,
   `socioeconomic_isolation_test.go`.
5. **Materializzazione e resume:** riusare `TestSimulatedSystemPlan`,
   `TestSimulatedSystemPlanResumeAndSelection`,
   `TestSimulatedSystemDocument` e `TestSimulatedSystemGeneration`
   (file in `core/` e `cmd/`). Confrontare documenti record-per-record per
   person canonica e fatti, record duplicati con ID osservati distinti,
   progress/checkpoint e output prima/dopo ripresa. Verificare i tre percorsi:
   `empty-v1` non deve acquisire asset; `generation-v3` di ripresa usa foto
   e dati base senza modello socioeconomico; `generation-v4` usa anche il
   modello. I vincoli correnti sono in `core/simulated_system.go`:23–27 e
   `core/simulated_system_generation.go`:177–203, 453–483, 514–532.
6. **Acquisizione:** aggiungere casi mirati per ordine delle origini, fallback
   dopo errore di trasporto o di consumo, scarto del candidato fallito,
   cache del successo e retry dell’errore; le suite resolver, modello e foto
   già contengono precedenti riutilizzabili. La rete deve terminare alla
   preparazione/acquisizione: generazione e ripresa dopo preparazione devono
   operare su stato immutabile senza accesso esterno.

Questo piano non è stato eseguito in questa ricognizione.

## G. Decisioni architetturali ancora necessarie

| Decisione / verifica aperta | Evidenza | Raccomandazione |
| --- | --- | --- |
| Unità di pubblicazione dei dati di base: sei documenti separati o un’acquisizione coerente della categoria. | Un singolo `parseLayer1ItalyData` costruisce tutte le componenti, e luoghi/CAP hanno vincolo uno-a-uno (`layer1_data.go`:237–328, 406–442). | Preferire un’unica acquisizione logica/preparazione con le stesse sei strutture runtime, anche se il packaging JSON non viene deciso ora. Riduce combinazioni di release e fallimenti parziali. |
| Come conservare gli identificatori SHA congelati quando il trasporto è JSON. | Gli hash attuali sono del CSV fisico; i test li verificano come componenti raw e i componenti entrano negli stream. | Per i valori italiani esistenti mantenere gli identificatori di compatibilità frozen indipendenti dai byte JSON; verificare con goldens, senza modificare contratti o output. |
| Semantica operativa delle origin fotografiche multiple. | Il catalogo usa fallback ordinato; URL WebP usa la prima origine. | Prima di cambiare codice, verificare se ogni origin pubblica lo stesso catalogo e gli stessi WebP o se è richiesto fallback per singola immagine. Quest’ultimo non esiste oggi. |
| Disponibilità e provenienza operativa degli asset sulle origin reali. | Guida e amendment dichiarano non verificata la disponibilità di modello su tutte le origin; questa ricognizione non ha fatto richieste di rete. Modello di deployment non trovato nel repository. | Confermare la pubblicazione degli asset, i relativi percorsi e l’equivalenza tra mirror prima di considerare chiusa l’acquisizione. |
| Stato documentale del producer D2-B2 v3. | Production report dichiara produzione canonica; file producer con hash corrispondente si autodefinisce noncanonico. | Chiarire la provenienza del modello pubblicato se la riproducibilità offline rientra nel prossimo incarico; non rigenerare né sostituire il modello in questa fase. |
| Condizioni di riuso futuro del modello socioeconomico tra nazioni. | Assi/ordinali, numero delle regioni e EUR sono incorporati nel runtime e nel contratto logico corrente. | Lasciare frozen `it-2026a`; decidere solo quando una concreta classificazione nazionale richiederà assi o semantiche incompatibili. |

**Stato del repository.** Branch e HEAD coincidono con quelli dichiarati.
All’inizio della ricognizione non c’erano modifiche tracked; il worktree
contiene molti file e directory untracked oltre alla guida intenzionale. Non
sono stati aperti o esplorati i contenuti di `.idea/`. Questo rapporto, creato
su richiesta nella root, è il solo nuovo file intenzionalmente aggiunto ed è
untracked; nessun altro file è stato modificato, spostato o rimosso.
