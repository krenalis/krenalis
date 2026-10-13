# Review tecnica della specifica JSON internazionale Synthetic World v1

Review indipendente del 12 ottobre 2026 sul branch `synthetic-world`, HEAD `50ed99476cf22a8137d1adcf8e2c82f7235e0ec6`.

Documento esaminato integralmente: [candidatura](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:1), 1.866 righe, 78.944 byte, SHA-256 `91b178892bf72086e0fbe7f46e9c5c7f53d1d97591b6115b0fb6cb577387851a`.

## A. Verdetto esecutivo

**PASS WITH NON_BLOCKING. Nessun BLOCKER riscontrato.**

I quattro schemi sono tecnicamente approvabili. La mappatura copre i dati logici italiani; il nuovo ingresso può ricostruire le strutture private già consumate dal motore Go e riutilizzare il compilatore socioeconomico. La geografia leggera conserva tutte le informazioni necessarie. I nuovi fingerprint possono utilizzare KFD1 e gli stream esistenti. Il descrittore e la verifica dei digest nel consumo di ciascun candidato sono compatibili con External Assets.

È opportuno precisare un'affermazione sulla migrazione fotografica: il discriminante di selezione attuale è `fakefacegen-v1`, mentre la release del catalogo esterno è `v1`. Copiare quest'ultima in `selection.version` è una scelta lecita della nuova baseline, ma non conserva il precedente discriminante dello stream.

L'approvazione degli schemi non certifica una conversione o un'implementazione ancora assenti. Come già prescritto dalla candidatura F.7 e I.3, l'ammissione del nuovo World e il caricamento internazionale richiedono un contratto di integrazione esplicito, versionato e circoscritto alla nuova baseline. I contratti frozen continuano a governare il percorso precedente.

## B. Verifiche effettuate

Sono stati letti gli `AGENTS.md` applicabili, la candidatura completa, i sei CSV, i parser e consumatori pertinenti, il modello canonico locale e i confini di acquisizione, identità e materializzazione applicativa. Guida, ricognizione e dossier sono presenti nel checkout e sono stati consultati come documentazione di supporto. Le conclusioni sul comportamento corrente derivano dal codice e dai contratti pertinenti.

Le verifiche sono statiche. Le ispezioni in memoria di CSV e modello hanno calcolato conteggi, hash e confronti documentali senza scrivere asset. Non sono stati eseguiti test, build, producer, generatori o migrazioni.

| Ambito | Esito | Evidenza principale |
|---|---|---|
| Dati di base | Completa per rappresentabilità e corrispondenza con i sei originali; conversione futura da verificare | B.2/H.1; `layer1_data.go`, sei CSV letti integralmente |
| Modello socioeconomico | Completa per copertura strutturale e numerica del modello disponibile; nessun nuovo asset da certificare | B.3/D/H.2; `socioeconomic_model_json.go`, `socioeconomic_model_compile.go`, modello canonico locale |
| Compatibilità internazionale | Completa sul contratto comune; esecuzione di altri paesi fuori ambito | Cardinalità derivate dalle tassonomie, otto consumatori d'età fissi, valuta esplicita; binding italiano separato |
| Proiezione italiana | Completa come definizione e fattibilità sui tipi privati; equivalenza dell'asset convertito non ancora dimostrabile | Tutte le tredici proprietà originali, otto mappe e sei branche ricostruibili direttamente |
| Determinismo | Completa sul disegno e sulle primitive; nuovi vettori e prove di esecuzione differiti | F.1–F.6; KFD1, factory immutabile, stream per indice/percorso, separazione delle dipendenze |
| Identità fotografiche | Completa sul confronto col codice; catalogo di produzione e nuovi digest non disponibili | F.5/H.3; `external_photo.go`, `layer1_photo.go`, snapshot e checkpoint |
| External Assets | Completa sul resolver e sull'ownership; nuove acquisizioni da verificare in implementazione | I.2; `Resolver.Resolve`, `Resolver.attempt`, i due source attuali |
| World e checkpoint | Completa sui vincoli correnti e sul confine necessario per v5; ripresa v5 non eseguibile ora | F.7/I.5; costruttori, policy v3/v4, checkpoint e worker |
| Completezza dei contratti | Completa per l'implementabilità dei quattro documenti; una precisazione non bloccante | B–D/F–I; obbligatorietà, oggetti chiusi, riferimenti, ordinamenti, errori e composizione |
| Impatto sul motore Go | Completa per i confini principali; nessuna riscrittura sostanziale necessaria | Strutture private riutilizzabili, compilatore unico, preparazione prima della generazione |

### B.1 Dati di base e coorti

La mappatura H.1 conserva ogni colonna. Le cardinalità effettive sono:

| Originale | Dati verificati | Ricostruzione nel Go |
|---|---|---|
| `age_weights.csv` | 55 righe, età 21–75; massa totale 41.485.820 | `ageRow`, `ages`, `ageWeights`; età rappresentabile come `int` nel profilo italiano, pesi `uint64` |
| `first_names.csv` | 800 righe; dieci gruppi da 80; per gruppo 20 pesi 4, 40 pesi 2, 20 pesi 1 | `firstNameGroup`, `firstNameRow`, vettori di pesi per gruppo |
| `last_names.csv` | 600 righe; 120 pesi 4, 240 pesi 2, 240 pesi 1 | `nameRow`, vettore di pesi |
| `places.csv` | 7.896 comuni; popolazioni positive, totale 58.942.828; venti regioni | `placeRow`, stessa slice e stessi bucket contigui da 128 |
| `postal_codes.csv` | 7.896 associazioni; copertura esatta dei comuni; liste da uno a otto CAP | `map[string][]string`, mantenendo l'ordine di ciascuna lista |
| `street_names.csv` | 400 frammenti; 80 pesi 4, 160 pesi 2, 160 pesi 1 | `nameRow` con il solo display utilizzato per la strada, vettore di pesi |

Gli SHA-256 dei sei file coincidono con i literal frozen in [layer1_data.go](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:29). Le strutture private e la costruzione dei bucket sono in [layer1_data.go](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:44). Non occorre modificare il campionamento pesato.

Le cinque coorti inclusive proposte coincidono con [cohortForYear](/Users/marco/src/krenalis/tools/fakedata/layer1_helpers.go:39): 1950–1964, 1965–1979, 1980–1989, 1990–1999 e 2000–2005. La formula di [World.birth](/Users/marco/src/krenalis/tools/fakedata/layer1.go:359) produce complessivamente date dal **1950-01-02 al 2005-01-01**. La coorte finale non deve essere accorciata al 2004: il giorno 2005-01-01 è generabile.

L'ordine fisico originale dei nomi è `female`, poi `male`, con le cinque coorti in ordine cronologico. La disposizione proposta dei gruppi permette di ricostruirlo; dentro ciascun gruppo va mantenuto l'ordine delle voci. `WeightedIndex` associa gli intervalli alle posizioni: un riordinamento dei nomi, anche a pesi invariati, cambierebbe l'associazione tra estrazione e nome. Questo vale anche per cognomi, strade, località e CAP. La candidatura lo prescrive correttamente in B.2, F.1 e H.1.

Codici ISTAT e CAP rimangono stringhe. Ad esempio `001001` e `09019` conservano gli zeri iniziali; `Agliè` conserva i byte UTF-8 del valore decodificato. `state_prov` passa al `code` dell'unico attributo provinciale/UTS italiano; `region` passa direttamente a `territory`. Non servono gerarchie territoriali ulteriori.

Unicità e atomi sono coerenti con la rettifica D1-B: la coorte appartiene alla chiave di unicità dei nomi propri; lo stesso display in coorti diverse è ammesso; le collisioni di `asciiAtom` non sono duplicati del display. Le regole comuni aggiungono unicità esatta, mentre il profilo italiano applica anche `D(name)`. Nessuna delle due richiede di deduplicare o alterare gli originali.

Il parser corrente verifica per i display soltanto non-vuoto, UTF-8 e assenza di spazi marginali; `validASCIIAtom` accetta lettere minuscole e trattini, senza cifre e senza vietare due trattini interni consecutivi. Il contratto D1-B ammette invece cifre negli atomi e comprime i separatori. Nei dati effettivi dei nomi e cognomi non sono presenti cifre né trattini consecutivi negli atomi. La maggiore precisione del nuovo validatore non obbliga a trasformare o rifiutare questi dati. Riferimenti: [validASCIIAtom e validDisplayName](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:456), [D1-B F.1–F.3](/Users/marco/src/krenalis/L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md:505), [rettifica G.2](/Users/marco/src/krenalis/L1-D1-B_Contract_Rectification.md:476). Il pin Unicode 16 richiede l'attenzione descritta in O-02.

### B.2 Proiezione socioeconomica

La proiezione è definita senza una lacuna numerica riscontrata. Le tredici proprietà originali sono coperte così:

| Proprietà originale | Proiezione prevista |
|---|---|
| `schema` | Literal `SW-L2-D2B2-logical-v0`, verificato attraverso `sourceModel.schema` |
| `companyCount` | Copia di 65.536 |
| `ages` | Otto rinomine esplicite; copia di `lowerBounds`, ricostruzione esatta di `compiled` |
| `companyIndustry` | Copia dei 14 pesi |
| `employeeCount` | Copia delle quattordici leggi nell'ordine Industry, 128 punti ciascuna |
| `education` | Copia degli 840 pesi |
| `primaryActivity` | Copia dei 6.720 pesi |
| `employeeJobFamily` | Copia dei 1.400 pesi |
| `employeeJobLevel` | Copia dei 1.680 pesi |
| `selfEmployedJobFamily` | Copia dei 1.400 pesi |
| `selfEmployedIndustry` | Copia dei 140 pesi |
| `employer` | Cap 4096; copia dei 140 fattori Industry e dei 16 fattori Size |
| `income` | Rinomina delle sei branche, copia di mappe e leggi; sola conversione prescritta `null → 65535` |

I pesi categorici sono complessivamente **12.194**. Alcuni superano il massimo di `int32`; la scelta di `U32` e dei tipi Go `uint32` è quindi necessaria, non decorativa. Pesi di base e popolazioni utilizzano invece `U64`, coerentemente con `italyData`.

Le otto partizioni rilevate nel modello locale coincidono con D.5:

| Consumatore internazionale | `lowerBounds` |
|---|---|
| `education` | 21, 25, 35, 45, 55, 65 |
| `activity` | 21, 25, 35, 45, 55, 60, 65, 70 |
| `employeeLevel` | 21, 25, 35, 45, 55, 65 |
| `selfEmployedIncome` | 21, 25, 35, 45, 55, 65 |
| `unemployedIncome` | 21, 25, 35, 45, 55, 65 |
| `studentIncome` | 21, 25, 30, 40, 55, 65 |
| `retiredIncome` | 21, 45, 55, 60, 65, 70 |
| `inactiveIncome` | 21, 25, 35, 45, 55, 65 |

Per ognuna, i 55 valori originali di `compiled` coincidono con l'indice dell'ultimo limite inferiore non maggiore dell'età. La ridondanza è quindi recuperabile integralmente: non viene scartata un'informazione indipendente. Non si devono unificare le otto partizioni.

Il registro D.1 coincide con gli indici in [socioeconomic_facts.go](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_facts.go:71): ultimo asse più veloce, outcome categorico più interno, nessuna trasposizione. Per Income employee la posizione è `(((R*10+F)*4+J)*14+I)*5+B`; per self-employed è `((A*20+R)*10+F)*14+I`; per gli altri quattro rami è `(A*7+E)*20+R`. Il registro rispetta i parent set dei contratti [4](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:448) e [5](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:523), e le forme di [D2-A](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md:396).

| Ramo originale | Slot della mappa | Leggi effettive | Punti per legge |
|---|---:|---:|---:|
| `e` | 56.000 | 16 | 385 |
| `s` | 16.800 | 12 | 385 |
| `u` | 840 | 8 | 385 |
| `t` | 840 | 8 | 385 |
| `r` | 840 | 8 | 385 |
| `i` | 840 | 8 | 385 |

Sono **76.160 slot**, **60 leggi Income** e **23.100 punti Income**. I soli **11.200 sentinel** sono ogni quinta posizione del ramo employee, corrispondente alla fascia Company zero. Nel nuovo trasporto diventano `null` esclusivamente lì. Nessuna legge viene assegnata a quella fascia; un supporto monetario `0` nelle leggi degli altri slot rimane un reddito reale. Un riferimento numerico `65535` non va trattato genericamente come `null` fuori dal profilo e dagli slot italiani prescritti.

Gli importi vengono copiati come euro interi `uint32`. Non c'è conversione monetaria, annualizzazione aggiuntiva o arrotondamento da applicare. Questo preserva [Contract 5, unità e zero](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:164).

L'ordine canonico delle leggi per prima occorrenza, l'unicità dei supporti, i pesi positivi delle leggi e il GCD unitario sono compatibili con [compileSocioeconomicModel](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_compile.go:168) e [compileSocioeconomicWeights](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_compile.go:235). Il compilatore rifiuta contenuti non canonici; non li normalizza silenziosamente.

Il percorso decoder internazionale → validazione del profilo → `socioeconomicModelJSON` → `compileSocioeconomicModel` è diretto e realizzabile nello stesso package. Non serve serializzare `logical-v0`, richiamarne il decoder o scrivere un secondo compilatore. Le verifiche oggi eseguite dal decoder precedente devono però essere mantenute al nuovo confine, come chiarito in O-04.

Companies e indici Employer restano una costruzione locale successiva. La formula `min(n,C) * I[F,I] * S[J,B+]`, i quaranta indici, l'ordine CompanyIndex crescente e il fallimento di un indice realizzato a totale zero coincidono con [NewSocioeconomicWorld](/Users/marco/src/krenalis/tools/fakedata/socioeconomic.go:49). Non servono nuovi tentativi con seed diversi, assegnazioni forzate o dipendenze dall'occupancy.

### B.3 Classificazioni e geografia internazionale

Le strutture comuni non fissano venti regioni, cinque coorti, sette livelli Education, quattordici Industry o cinque SizeBand. Questi numeri appartengono al profilo italiano e al runtime italiano. Gli assi sono una famiglia chiusa intenzionale; le loro cardinalità vengono dalle categorie ordinate e dalle partizioni dichiarate. La fascia zero distinta, le sei branche e gli otto consumatori d'età sono vincoli della famiglia modellistica scelta, non un'imposizione nascosta della classificazione italiana.

Valuta e precisione sono dichiarate; i codici territoriali e postali comuni non richiedono cifre o lunghezze italiane; `administrative` ammette attributi autonomi senza inventare una gerarchia. L'intervallo italiano 21–75 e i frammenti stradali sono vincoli del profilo, mentre il formato ammette dominio e lessico differenti.

Il territorio socioeconomico è autonomo nel modello. La compilazione italiana può avvenire senza acquisire il documento base. Prima della costruzione si confrontano `territorySystem` e `dimensions.territory.semanticId`, quindi si risolve esattamente ogni codice delle località. La preparazione può ricondurre direttamente il codice alla stessa label e al medesimo ordinale regionale usati da [socioeconomicRegions](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_facts.go:42). Provincia e ISTAT non entrano nel DAG.

`semanticId` è verificabile mediante un binding locale chiuso: ruolo, ID semantico, codici, ordine, branche e soglie previsti. Non occorre un registry remoto. Il profilo italiano deve rifiutare una tassonomia sconosciuta anche quando le sue cardinalità coincidono. La validità strutturale di un altro paese non implica l'eseguibilità dal generatore italiano.

Un riferimento mancante dentro un documento è un errore autonomo; due documenti validi con sistemi territoriali o ancoraggi incompatibili producono un errore di composizione. La distinzione G.1/G.3 è implementabile senza invalidare il modello autonomo già acquisito.

### B.4 Determinismo e identità

Le sei proiezioni F.4 sono chiuse e coprono i dati consumati da ciascuna famiglia. I domini sono distinti e ammessi dalla grammatica KFD1; le descrizioni delle categorie sono escluse dalla proiezione nominale come prescritto. La normalizzazione degli array è limitata al catalogo fotografico. Per gli altri array l'ordine è parte dei valori logici.

| Fingerprint logico | Componente bytes mantenuto | Consumatore corrente |
|---|---|---|
| Età | `age-data-sha256` | `identity/birth/age` |
| Nomi propri | `first-name-data-sha256` | `identity/name/first` |
| Cognomi | `last-name-data-sha256` | `identity/name/last` |
| Località | `places-data-sha256` | `geography/place/uniform`, `geography/place/population` |
| CAP | `postal-data-sha256` | `geography/place/postal` |
| Strade | `street-name-data-sha256` | `geography/address/fragment` |

I due nomi geografici lasciati al riscontro nella candidatura sono quelli letterali sopra riportati: [layer1.go](/Users/marco/src/krenalis/tools/fakedata/layer1.go:42), [World.municipality](/Users/marco/src/krenalis/tools/fakedata/layer1.go:469). Il ramo 95/5 rimane `geography/place/mixture`, senza dipendenza dall'intero pacchetto. Genere, giorno di nascita, tipo di strada e civico conservano i propri stream.

KFD1 ordina i componenti per nome, distingue stringhe/interi/bytes e codifica le lunghezze: [component.go](/Users/marco/src/krenalis/tools/fakedata/component.go:123). La factory deriva ogni RNG nuovamente da radice, tipo/indice dell'entità, percorso e dipendenze: [stream.go](/Users/marco/src/krenalis/tools/fakedata/stream.go:25). I nuovi hash non richiedono modifiche a PRNG, rejection sampling o indirizzamento. Ordine delle richieste, concorrenza e batch non entrano nella derivazione.

La radice resta `world-v1` e WorldSeed. `person-v2` distingue la nuova specifica nella composizione dell'identità del World; `layer1-json-v1` identifica il nuovo protocollo dei dati attraverso i suoi domini e valori di dipendenza. Non occorre sostituire la radice o inserire l'identità del pacchetto negli stream L2. Il compilatore prepara ancora i quindici fingerprint dai soli sottoalberi frozen e dalle relative mappe d'età, poi [socioeconomic_sampling.go](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_sampling.go:118) li usa con la factory effettiva del World base.

Le quattro identità socioeconomiche vanno tenute distinte:

1. SHA-256 dei byte dell'originale: attestazione fisica del file locale.
2. `M_all = SHA256(C(L))`: digest dei valori canonici ricostruiti dal Go.
3. `D_socioeconomic`: digest KFD1/C1 del nuovo documento completo, compresi metadati internazionali.
4. `SocioeconomicWorldID`: combinazione frozen di WorldID base effettivo, `M_all`, ancoraggio e `socioeconomic-v1`.

Per [D2B2-logical-assets-v0.json](/Users/marco/src/krenalis/D2B2-logical-assets-v0.json:1), 685.312 byte, è stata verificata in memoria l'uguaglianza dei byte originali con la serializzazione canonica documentata in [protocollo §4.1](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md:66). Lo SHA-256 è `f049103241e6e43ffb5dd4a1247b64b2d0ca1bff787950f25284b5b435fd21a8`. Ciò conferma l'affermazione F.6 nei suoi limiti documentali. Non è stata eseguita `canonicalModel()` né verificata una proiezione da un asset internazionale futuro.

Il WorldID base deve usare la release base e continuare a escludere foto e L2. [socioeconomicWorldIdentity](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_identity.go:227) non riceve snapshot o catalogo. Pertanto una modifica esclusivamente fotografica può cambiare lo SnapshotID e lasciare invariato il SocioeconomicWorldID. PersonID resta la coordinata reversibile nel namespace: [identity.go](/Users/marco/src/krenalis/tools/fakedata/identity.go:111). Non deve essere trasformato in un hash del contenuto della persona; due baseline possono avere gli stessi PersonID e fatti differenti, senza retrocompatibilità degli output CSV.

### B.5 Fotografie, acquisizione e ripresa

La separazione F.5 è coerente con l'algoritmo corrente. Dopo ordinamento per ID, il selettore può mantenere i due passaggi per distanza minima e la scelta uniforme nello stream `photo/select`: [FaceCatalog.selectPhoto](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:287).

Nel nuovo percorso le due responsabilità sono:

| Uso | Valore |
|---|---|
| Dipendenza casuale della selezione | `F_photo`, con `selection.version` come discriminante |
| Attestazione del documento rispetto al descrittore | Digest completo `D_photos` |
| Identità fotografica dello Snapshot | Digest completo `D_photos` |
| `CatalogSHA256` del checkpoint v5 e metadati dell'asset | Digest completo `D_photos` |
| Identità socioeconomica | Nessun digest fotografico |

Oggi un solo checksum serve questi primi usi: viene copiato dal `selectionSeed` per il catalogo esterno, entra nelle dipendenze, nello Snapshot e nel checkpoint. I punti sono stati riscontrati in [external_photo.go](/Users/marco/src/krenalis/tools/fakedata/external_photo.go:195), [newWorld](/Users/marco/src/krenalis/tools/fakedata/layer1.go:199), [FaceCatalog.Checksum](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:226), [inizializzazione e controllo del checkpoint](/Users/marco/src/krenalis/core/simulated_system_generation.go:434). Il percorso locale usa inoltre il checksum per `/photos/<checksum>/...` e per i metadati verificati delle immagini: [layer1_photo.go](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:189), [publicPhotoPath](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:694). Queste semantiche precedenti non devono essere reinterpretate.

Modificare un candidato cambia il fingerprint della selezione; cambiare soltanto una rendition o il suo percorso lascia il fingerprint invariato e cambia il digest completo. Quindi il PhotoID canonico rimane uguale, mentre snapshot e controllo del catalogo nel checkpoint cambiano. La ripresa che rileva un catalogo completo diverso può correttamente fallire anche se il PhotoID sarebbe uguale. Le osservazioni Source hanno una dipendenza dallo Snapshot, descritta in O-03.

Il resolver conserva le origin in ordine; qualunque errore del consumer provoca fallback. Verificare il digest atteso nella callback permette a un JSON valido ma diverso dal riferimento di essere scartato prima di accettare la origin. [Resolver.attempt](/Users/marco/src/krenalis/tools/externalassets/resolver.go:111) completa anche la lettura dopo il consumer: la cache va pubblicata soltanto al ritorno riuscito di `Resolve`. Questo è già il comportamento dei source attuali.

Gli asset dei tre ruoli possono arrivare da origin differenti se i contenuti logici corrispondono ai digest del descrittore. Non si mescolano release incompatibili soltanto perché il resolver effettua fallback separatamente: la combinazione viene vincolata dai tre riferimenti e dalle validazioni incrociate. Se i digest corrispondono ma la combinazione è incompatibile, fallisce la preparazione della combinazione; non si deve cercare un asset diverso in violazione del digest.

Successo, attese concorrenti e retry dopo fallimento sono riutilizzabili dai source esistenti. Il resolver non contiene una cache per percorso; l'isolamento dei nuovi riferimenti va costruito nell'owner applicativo, con la misura circoscritta descritta in O-05.

Startup configura i source senza acquisire gli asset remoti: [cmd/server.go](/Users/marco/src/krenalis/cmd/server.go:61). Oggi la creazione non vuota acquisisce il catalogo per inizializzare il checkpoint; il worker acquisisce L2 soltanto per v4. Sono entrambi confini da collegare alla preparazione v5 quando necessario. `empty-v1` e i sistemi Available non devono attivare quel percorso. Il worker seleziona solo Preparing/Deleting: [accountsToProcess](/Users/marco/src/krenalis/core/simulated_system_generation.go:85).

Il piano corrente accetta soltanto v3/v4 e costruisce sempre il World dei CSV embedded: [newSimulatedSystemPlan](/Users/marco/src/krenalis/core/simulated_system_generation.go:453). Aggiungere soltanto la stringa `generation-v5` a una condizione non basta. V5 deve selezionare i dati acquisiti, il nuovo ingresso World e i valori di versione/checksum pertinenti, conservando il ramo precedente per v3/v4. Le coordinate di ripresa e il conteggio dei record possono restare quelli esistenti.

La materializzazione usa già `Asset(..., PhotoSize256)` e `photoBaseURL.JoinPath(asset.Path)`: [record](/Users/marco/src/krenalis/core/simulated_system_generation.go:591). Il nuovo catalogo può fornire il percorso dichiarato, verificando la rendition 256×256 per tutti i candidati. Gli URL WebP restano sulla prima origin configurata, come stabilisce [cmd/server.go](/Users/marco/src/krenalis/cmd/server.go:66); il fallback dei metadati non implica fallback delle immagini.

Non occorrono controlli di SocioeconomicWorldID, `M_all`, digest L2 o identità del pacchetto nei checkpoint. La politica di immutabilità fra processi resta distinta dai controlli di ripresa e dai digest dichiarati: [guida §5.2](/Users/marco/src/krenalis/SYNTHETIC_WORLD_GUIDA_PROGETTO.md:143), candidatura I.5.

### B.6 Completezza dei quattro contratti

Le convenzioni B.1 distinguono proprietà obbligatorie e facoltative, chiudono gli oggetti e non attribuiscono default ai campi assenti. L'unico oggetto con chiavi libere è la mappa postale. `null` non equivale ad assenza: è ammesso soltanto negli slot Income employee prescritti. Gli interi hanno grammatica e limiti espliciti; `U32`, `U64` e pesi positivi sono distinti, e somme/prodotti devono essere verificati prima dell'uso. Stringhe e codici non vengono ripuliti o normalizzati dal decoder.

| Contratto | Completezza verificata |
|---|---|
| `sw-base-data-v1` | Metadati, genere, età, coorti, gruppi nominali, cognomi, geografia, mappa postale e lessico stradale obbligatori. `asciiAtom` facoltativo nel comune e obbligatorio in Italia; nessuna generazione implicita. Gli attributi amministrativi richiedono almeno codice o nome. Riferimenti, unicità, copertura dei gruppi e ordinamenti sono definiti |
| `sw-socioeconomic-data-v1` | Dominio, sette dimensioni, otto partizioni, unità e tutti i modelli obbligatori. `sourceModel` facoltativo nel comune e obbligatorio nel profilo italiano. Assi, forme dense, outcome, branche, riferimenti alle leggi e sentinel sono definiti; nessuna riga mancante viene supplita con un fallback |
| `sw-photo-catalog-v1` | Tassonomia, versione/seed e candidati obbligatori; ogni candidato ha rendition dichiarate. Integrità facoltativa, con hash e byte obbligatori quando presente. Unicità di ID e dimensioni, coerenza dei percorsi ripetuti e ordinamento normalizzato sono espliciti. Gli hash assenti non vengono inventati |
| `sw-country-release-v1` | Paese, release e tutti e tre i riferimenti obbligatori; ogni riferimento dichiara percorso e digest logico atteso. Il ruolo fissa lo schema ammesso, i percorsi sono relativi alla origin e le subrelease possono differire. I nomi dei file non determinano profilo o compatibilità |

Il comportamento degli errori è sufficientemente definito: rifiuto del candidato autonomo, fallback del resolver, fallimento della preparazione se nessuna origin riesce, rifiuto della combinazione incompatibile e nessun World parzialmente valido. Collocazione delle funzioni, nomi dei tipi privati e struttura circoscritta della cache sono dettagli implementativi. Non emergono altre regole semantiche che Codex debba inventare per implementare i quattro decoder.

## C. Findings

I riferimenti alla candidatura identificano sezioni e righe del documento esaminato. Nessun rilievo è classificato BLOCKER.

### N-01 NON_BLOCKING Precisare il discriminante fotografico precedente

- **Clausola:** B.4, [riga 486](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:486); H.3, [riga 1713](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:1713); F.5, righe 1484–1491.
- **Evidenza:** `decodeExternalPhotoCatalog`, [external_photo.go:203](/Users/marco/src/krenalis/tools/fakedata/external_photo.go:203), costruisce `face-catalog-version` da `fakefacegenV1SourceVersion`; [fakefacegen.go:25](/Users/marco/src/krenalis/tools/fakedata/fakefacegen.go:25) ne fissa il valore a `fakefacegen-v1`. La release esterna è separatamente `v1`.
- **Comportamento verificato:** lo stream attuale non usa la release `v1` come componente di versione.
- **Rischio concreto:** interpretare la copia `release → selection.version` come conservazione del vecchio discriminante rende inesatta la descrizione della migrazione.
- **Conseguenza:** si possono costruire aspettative errate sui componenti casuali conservati. Non impedisce l'implementazione: F.5 e H.3 dichiarano già intenzionale una nuova baseline, e la compatibilità con gli output precedenti non è richiesta.
- **Correzione minima:** chiarire che copiare `release` sceglie esplicitamente il nuovo discriminante e non conserva il precedente `fakefacegen-v1`.
- **Destinazione:** precisazione della specifica, senza cambiare algoritmo o asset precedenti. Test dei componenti nel nuovo percorso durante l'implementazione.

### O-01 OBSERVATION C1 richiede un adattamento locale delle utility JSON

- **Clausola:** F.1/F.2, [righe 1259–1343](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:1259); K, riga 1855.
- **Evidenza:** [tools/json/functions.go:53](/Users/marco/src/krenalis/tools/json/functions.go:53) offre `Canonicalize` RFC 8785; [tools/json/decoder.go:22](/Users/marco/src/krenalis/tools/json/decoder.go:22) usa `jsontext`; [tools/json/buffer.go:144](/Users/marco/src/krenalis/tools/json/buffer.go:144) offre `EncodeSorted`.
- **Comportamento verificato:** la canonicalizzazione RFC 8785 della libreria standard tratta i numeri come IEEE 754 e ordina i nomi per UTF-16. La catena locale è `tools/json.Canonicalize → jsontext.Value.Canonicalize`, con confronto `jsonwire.CompareUTF16` in `/usr/local/go/src/encoding/json/jsontext/value.go:273`. Non coincide con C1 per tutti gli `U64` e le chiavi Unicode. Il decoder rigoroso già disponibile rifiuta duplicati decodificati e UTF-8 invalido; i nuovi decoder devono aggiungere oggetti chiusi, presenza, grammatica intera, limiti e i soli `null` consentiti.
- **Rischio concreto:** usare `Canonicalize` come C1 può arrotondare interi oltre 2^53 o produrre un ordinamento diverso; usare indiscriminatamente `EncodeSorted` su struct non garantisce il riordino dei loro campi. Inoltre `EncodeSorted` elimina il LF finale e rappresenta slice nil come `null`.
- **Conseguenza:** digest differenti a parità di contratto oppure `null` illegali, per esempio al posto di `axes: []`.
- **Misura minima:** riutilizzare decoder, quoting e codifica esatta dei tipi interi; emettere i campi delle strutture chiuse in ordine prescritto e ordinare per stringa UTF-8 le mappe a chiavi valide; controllare slice vuote/non-nil e aggiungere esattamente un LF. Le mappe tipizzate di stringhe nella modalità deterministica esistente hanno l'ordine utile a C1. Non è necessaria una nuova libreria generale di canonicalizzazione.
- **Destinazione:** implementazione locale e vettori C1. Nessuna correzione sostanziale della specifica: la regola di byte è già chiara.

### O-02 OBSERVATION Il pin Unicode 16 non coincide con il default vendored

- **Clausola:** B.2, [righe 203–209](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:203), G.2 e K.
- **Evidenza:** D1-B [F.1](/Users/marco/src/krenalis/L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md:511) e rettifica [G.2](/Users/marco/src/krenalis/L1-D1-B_Contract_Rectification.md:481) fissano Unicode 16. La catena vendored `cases.Fold → makeFold → caseFolder.Transform → foldFull` è in [cases.go:93](/Users/marco/src/krenalis/vendor/golang.org/x/text/cases/cases.go:93), [fold.go:12](/Users/marco/src/krenalis/vendor/golang.org/x/text/cases/fold.go:12), [context.go:334](/Users/marco/src/krenalis/vendor/golang.org/x/text/cases/context.go:334). Per Go 1.27 le [tabelle cases](/Users/marco/src/krenalis/vendor/golang.org/x/text/cases/tables17.0.0.go:3) e [norm](/Users/marco/src/krenalis/vendor/golang.org/x/text/unicode/norm/tables17.0.0.go:3) dichiarano Unicode 17.
- **Comportamento verificato:** il parser CSV non esegue già `D`; non esiste quindi un validatore runtime Unicode 16 pronto da richiamare. Negli originali ispezionati non emerge un dato che debba essere trasformato per la nuova rappresentazione.
- **Rischio concreto:** utilizzare la versione implicita delle tabelle e dichiararla Unicode 16, senza verificarne la conformità sul repertorio ammesso dal profilo.
- **Conseguenza:** la validazione di unicità e ordinamento potrebbe divergere dal pin per dati successivi o caratteri diversi, pur restando corretta sui casi più semplici.
- **Misura minima:** rendere esplicita e verificare la conformità a Unicode 16 della validazione del profilo. Per la release italiana fissata è sufficiente dimostrare tale conformità sul repertorio ammesso, senza rigenerare gli atomi o normalizzare i display caricati. Non occorre imporre Unicode 16 a tutti i testi dei formati internazionali.
- **Destinazione:** implementazione del validatore e verifica della release; nessuna modifica al pin frozen. I controlli con la UCD locale effettuati nella review non sono una certificazione Unicode 16.

### O-03 OBSERVATION World e Source hanno controlli e responsabilità di versione distinti

- **Clausola:** F.7, [righe 1548–1565](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:1548); I.3 e I.5.
- **Evidenza:** [newWorld:154](/Users/marco/src/krenalis/tools/fakedata/layer1.go:154) acquisisce sempre i CSV embedded e usa le costanti correnti per WorldID; [NewSocioeconomicWorld:28](/Users/marco/src/krenalis/tools/fakedata/socioeconomic.go:28) controlla inizializzazione e forma dell'identità ma non contiene una whitelist nominale `person-v1`; [newSourceWorld:84](/Users/marco/src/krenalis/tools/fakedata/source_simulation.go:84) accetta IT soltanto con `CountryShare.Version == MarketDataVersion`, oggi `it-2026a`. [newSimulatedSystemPlan:472](/Users/marco/src/krenalis/core/simulated_system_generation.go:472) verifica policy e versioni precedenti.
- **Comportamento verificato:** il nuovo World non è ottenibile cambiando solo una configurazione pubblica. L2 potrebbe superare i controlli meccanici correnti, ma [Contract 1 §7](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:217) prescrive la provenienza da `NewWorld`; l'amendment non autorizza altre baseline. SourceWorld include lo SnapshotID nei propri componenti.
- **Rischio concreto:** sostituire globalmente `PersonSpecVersion`/`MarketDataVersion`, ammettere v5 nel ramo precedente o passare automaticamente la nuova `base.release` come `CountryShare.Version`.
- **Conseguenza:** alterazione o mancata ripresa dei percorsi v3/v4; rifiuto di IT da SourceWorld; ammissione normativa implicita della nuova base.
- **Misura minima:** un nuovo ingresso World e un ramo di preparazione/checkpoint v5 espliciti, con ammissione L2 autorizzata dal contratto di integrazione già richiesto. Source Simulation può conservare il binding IT corrente `it-2026a`: lo SnapshotID distingue già la nuova baseline. Una futura scelta di cambiare quel binding richiederebbe soltanto un'ammissione circoscritta nel costruttore, non la riscrittura di Source/Oracle.
- **Destinazione:** nuovo contratto di integrazione e wiring. Nessuna modifica dei frozen. La modifica del catalogo completo cambia lo Snapshot e può quindi cambiare ID/osservazioni Source, anche quando il PhotoID canonico resta uguale: F.5 non promette l'invarianza di tali osservazioni.

### O-04 OBSERVATION L'adattatore numerico deve conservare le validazioni oggi nel decoder

- **Clausola:** B.3, D, G.2 e [I.3, righe 1770–1777](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:1770).
- **Evidenza:** [socioeconomicModelJSON](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_json.go:25) è direttamente costruibile da nuovi file dello stesso package. [decodeSocioeconomicModel:298](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_json.go:298) verifica limiti e restringimenti prima della compilazione. [compileSocioeconomicModel:37](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_compile.go:37) assume i limiti del decoder; le verifiche complete di forme, cumulative e leggi avvengono nel compilatore.
- **Comportamento verificato:** il compilatore non ripete autonomamente tutti i massimi del decoder, per esempio cap/fattori prima della conversione a `uint16`, cardinalità massime delle leggi e numero massimo di bucket.
- **Rischio concreto:** bypassare il vecchio decoder e considerare sufficiente la sola chiamata al compilatore.
- **Conseguenza:** un ingresso internazionale fuori profilo potrebbe essere ristretto o ammesso in modo diverso dall'ingresso numerico frozen.
- **Misura minima:** validare nel nuovo decoder/profilo le stesse precondizioni prima di popolare i tipi privati e di restringere i valori; demandare al compilatore esistente la compilazione e i controlli numerici che già possiede. La ricostruzione di 440 indici d'età e la conversione degli slot `null` sono adattamento della rappresentazione, non un secondo compilatore.
- **Destinazione:** implementazione e prove di rifiuto mirate. La candidatura già assegna questo lavoro ai profili e all'adattatore; nessuna correzione dello schema necessaria.

### O-05 OBSERVATION La cache attuale appartiene al source e non al percorso nel resolver

- **Clausola:** [I.2, righe 1754–1760](/Users/marco/src/krenalis/SYNTHETIC_WORLD_INTERNATIONAL_JSON_SPEC_v1_CANDIDATE.md:1754), G.3.
- **Evidenza:** [Resolver](/Users/marco/src/krenalis/tools/externalassets/resolver.go:25) contiene soltanto origin/client. [SocioeconomicModelSource.Load](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model.go:67) e [FaceCatalogSource.Load](/Users/marco/src/krenalis/tools/fakedata/external_photo.go:49) conservano ciascuno un solo successo nell'istanza e coordinano i caricamenti con mutex/canale.
- **Comportamento verificato:** oggi il percorso fisso è implicito nell'identità del source. Non esiste una cache globale da correggere o un download da spostare fuori dai campionatori.
- **Rischio concreto:** riutilizzare il medesimo risultato per due riferimenti con digest attesi differenti perché il percorso coincide, oppure pubblicare il candidato prima che `Resolve` finisca di controllare il trasporto.
- **Conseguenza:** l'owner potrebbe restituire un asset di un'altra combinazione o un risultato non definitivamente acquisito.
- **Misura minima:** possedere un risultato immutabile per il riferimento completo. Un owner legato a una release fissa può incorporarvi ruolo/schema/percorso/digest atteso; se gestisce più riferimenti, basta una chiave con quegli elementi nell'ambito delle origin possedute. Verificare il digest nella callback, pubblicare dopo `Resolve`, mantenere retry e coordinamento concorrente. Un errore di composizione non deve cancellare un modello autonomamente valido.
- **Destinazione:** implementazione circoscritta del source. Nessun framework generale di cache, resolver parallelo o nuova variabile d'ambiente.

## D. Correzioni minime eventualmente necessarie

**Nessuna correzione bloccante.** Per N-01 è raccomandata questa sola precisazione testuale.

In B.4, sostituire la frase finale sul precedente valore `release` con:

> Nel runtime precedente il discriminante di selezione è `fakefacegen-v1`, distinto dalla release `v1`. La migrazione descritta in H.3 copia la release in `selection.version` come scelta esplicita della nuova baseline; questa copia non conserva il componente di versione dello stream precedente.

In H.3, sostituire la descrizione della riga `release` con:

> Copiato in `selection.version` come scelta esplicita del nuovo discriminante, distinta dal precedente discriminante runtime `fakefacegen-v1`.

Le formule F.4/F.5, la geografia e la proiezione L2 non richiedono riscrittura. Gli altri rilievi descrivono lavoro implementativo già compreso nell'architettura. Il contratto di integrazione successivo deve esplicitare la nuova ammissione senza emendare retroattivamente i documenti frozen.

## E. Impatto implementativo verificato

| Confine nel repository | Modifica effettivamente necessaria |
|---|---|
| `tools/fakedata`, nuovi decoder/source | Quattro documenti chiusi, C1, digest attesi, profili e preparazione coordinata. Riutilizzo di `tools/json` ed `externalassets.Resolver` |
| `layer1_data.go`, strutture private | Preparazione equivalente a `italyData`, medesime slice/mappe/bucket, sei fingerprint logici nel solo nuovo percorso |
| `layer1.go`, costruzione | Ingresso con dati già acquisiti, versioni e release base della nuova baseline; mantenimento dell'ingresso precedente |
| Tipi `socioeconomicModelJSON` e adattatore italiano | Trasferimento dei valori, ricostruzione di `compiled`, conversione selettiva dei sentinel; controllo delle precondizioni del compilatore |
| `socioeconomic.go`, ammissione | Riconoscimento esplicito della nuova provenienza; riutilizzo di costruzione Companies/Employer e modello immutabile |
| `external_photo.go` / `layer1_photo.go` | Ingresso del nuovo catalogo, normalizzazione ID/rendition, due identità fotografiche e percorsi dichiarati. Le restrizioni del vecchio validatore, fra cui età 21–75 e le cinque `PhotoSize`, appartengono al consumatore precedente, non allo schema comune |
| `cmd/server.go`, configurazione Core | Owner del nuovo source con le origin esistenti, configurato senza download indiscriminato allo startup |
| `core/simulated_system.go` | Preparazione v5 richiesta dalla creazione non vuota e inizializzazione coerente del checkpoint |
| `core/simulated_system_generation.go` | Dispatch separato per empty/v3/v4/v5; costruzione v5 da dati esterni; checkpoint fotografico col digest completo; materializzazione dal percorso della rendition |

Non emerge la necessità di riscrivere nascita, selezione pesata, miscela geografica 95/5, bucket, contatti, indirizzi, `selectPhoto`, campionatori L2, formule di indicizzazione, DAG, Companies, Employer, primitive casuali o codec degli ID. Il compilatore numerico e la canonicalizzazione frozen L2 rimangono l'autorità riutilizzata. Eventuali condivisioni di codice di preparazione devono conservare esattamente il comportamento dell'ingresso precedente.

Source Simulation, Oracle, duplicati, Source/Destination e i sei campi socioeconomici applicativi possono rimanere invariati nelle loro semantiche. L'unico dettaglio di binding Source riscontrato è quello di O-03, risolvibile senza cambiare gli algoritmi. Non occorrono generatori di altri paesi, plugin nazionali, servizi immagini o controlli socioeconomici nei checkpoint.

## F. Verifiche da eseguire durante l'implementazione

1. **Conversione dei sei CSV.** Confronto integrale campo per campo e record per record, preservando stringhe, atomi, pesi, gruppi e ordinamenti. Verificare cardinalità e quote sopra riportate, coorti e date estreme, tutte le associazioni CAP e la ricostruzione di `italyData`/bucket. Test Unicode 16 del profilo, cifre e separatori degli atomi, collisioni consentite fra atomi e ripetizioni consentite fra coorti. Nessuna ricurazione o normalizzazione sostitutiva.
2. **Proiezione L2 completa.** Dal nuovo asset italiano ricostruire direttamente l'ingresso tipizzato e confrontare tutte le tredici proprietà con il modello canonico, incluse 440 posizioni d'età, 12.194 pesi categorici, 14 leggi EmployeeCount da 128 punti, cap/fattori, 76.160 slot e 60 leggi Income da 385 punti. Controllare i soli 11.200 `null ↔ 65535`, zero monetario e ordine per prima occorrenza. Verificare rifiuto di cardinalità, ordinali, unità e restringimenti fuori profilo.
3. **Identità canoniche.** Confrontare i byte prodotti da `canonicalModel()` prima/dopo l'adattamento con l'originale, `M_all` e tutti i quindici fingerprint frozen. Verificare separatamente i quattro digest internazionali, l'identità del pacchetto e le sei proiezioni L1. Vettori C1 con interi oltre 2^53 e fino a `U64` massimo dove ammesso, chiavi duplicate dopo escape anche annidate, Unicode fuori BMP, confronto non UTF-16, UTF-8 invalido, surrogate isolate, slash, U+2028/U+2029, controlli, LF, campi assenti, array vuoti e soli `null` consentiti; rifiutare frazioni, esponenti, negativi, `-0`, booleani e documenti multipli.
4. **Determinismo nuovo e isolamento.** Stesso World v5 e medesimi asset logici con ordini delle richieste, goroutine, subset, batch e riavvii differenti devono produrre gli stessi risultati. Cambiare una sola famiglia e verificare quali fingerprint/stream cambiano, distinguendo gli effetti funzionali discendenti. Variazioni di origin, compressione, whitespace e ordine delle proprietà non devono cambiare gli output. Conservare i test/vettori precedenti per i percorsi precedenti, senza sostituirne i goldens con quelli v5.
5. **External Assets.** Fallback sia di trasporto sia di consumo; soprattutto primo candidato strutturalmente valido ma con digest errato e secondo corretto. Verificare pubblicazione solo dopo il ritorno riuscito del resolver, retry dopo fallimento, attese concorrenti, cache del successo e isolamento di due digest attesi sullo stesso percorso. Combinazioni autonomamente valide ma incompatibili devono fallire prima del World. Nessun fallback ai CSV nel percorso v5, nessuna rete per `Person`/`Facts`/`Company`, nessuna nuova acquisizione per empty/Available o percorsi precedenti che non la richiedono.
6. **Selezione fotografica.** Normalizzazione indipendente dall'ordine fisico di faces/rendition, rifiuto dei duplicati, candidati di pari distanza con ordine ID e scelta uniforme esistenti. Mutare candidato/seed/versione deve cambiare `F_photo`; mutare soltanto path/rendition/integrità/release deve lasciare invariato il PhotoID canonico e cambiare il digest completo. Verificare l'uso dei due valori nei componenti corretti, negli Snapshot e nei checkpoint. Non usare le osservazioni Source come prova dell'invarianza del PhotoID.
7. **Ripresa e materializzazione.** Accesso diretto e ripresa v5 devono produrre gli stessi documenti e record Oracle, inclusi duplicati e interruzioni ai confini dei batch. V3/v4 devono rimanere sui propri rami e rifiutare input di baseline diversa; Available non va rigenerato. Verificare checkpoint fotografico, versioni, coordinate e contatore esistenti; assenza di nuovi controlli L2. URL 256×256 dalla prima origin e dal path dichiarato anche quando i metadati provengono da fallback. Verificare i sei fatti applicativi, zero Income e assenza dei campi non applicabili.

Queste verifiche estendono evidenze già presenti, fra cui [dipendenze Layer 1](/Users/marco/src/krenalis/tools/fakedata/layer1_data_test.go:15), [semantica geografica](/Users/marco/src/krenalis/tools/fakedata/layer1_geography_test.go:15), [vettori L2](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_protocol_test.go:99), [fallback](/Users/marco/src/krenalis/tools/externalassets/resolver_test.go:184) e [materializzazione fotografica](/Users/marco/src/krenalis/core/simulated_system_external_photo_test.go:82). La review ne ha letto il codice, senza eseguirli.

## G. Limiti della review

- I quattro asset internazionali di produzione non sono stati prodotti o acquisiti. È verificata la fattibilità della mappatura, non l'equivalenza materiale di una conversione futura.
- Il modello locale originale è disponibile e il suo hash/forma canonica documentata sono stati riscontrati. Non sono stati eseguiti compilatore Go, calcolo dei quindici nuovi risultati attraverso l'adattatore, campionamento o ripresa applicativa.
- Le verifiche di normalizzazione/case folding effettuabili con la UCD locale non certificano il pin Unicode 16. Occorre la verifica circoscritta indicata in O-02.
- Il catalogo fotografico reale pubblicato, il suo seed e la disponibilità/integrità dei WebP sulle origin non sono stati verificati. Le fixture e gli esempi della candidatura non sono stati trattati come asset di produzione.
- Disponibilità e immutabilità delle release fra processi restano obblighi di pubblicazione. I digest del descrittore impediscono di accettare documenti differenti da quelli dichiarati; non attestano da soli l'immutabilità del descrittore fra due riavvii.
- La capacità internazionale è valutata sulla struttura e sui binding. Non è certificato un generatore di un altro paese e non è stata progettata un'implementazione USA.

## H. Stato finale del repository

- Branch: `synthetic-world`.
- HEAD: `50ed99476cf22a8137d1adcf8e2c82f7235e0ec6`, coincidente con lo stato noto.
- File tracked: nessuna modifica o staging riscontrati nel perimetro esaminato.
- All'inizio della review corrente risultavano 29.244 file untracked, escludendo `.idea/**`; il file candidato era già presente. È stato aggiunto soltanto questo rapporto, come autorizzato dall'ultima istruzione dell'utente.
- Guida, ricognizione, dossier, candidatura e modello canonico conservano gli SHA-256 rilevati prima della scrittura del rapporto. È stato verificato anche il mantenimento di 38 file di supporto/protezione, comprese le evidenze D2-B2 presenti nella root, senza eseguirle o modificarle.
- Lo stato Git finale, escluso il nuovo rapporto, coincide con quello iniziale. Il digest della rappresentazione `status --porcelain=v1 -z --untracked-files=all`, nel perimetro con esclusione `.idea/**` e senza il rapporto, è `5e3715f940bd571a72e724711714b12f8d4862c714f8e99e3868b77ceaccc3a7`.
- Nessuna modifica a codice, CSV, JSON, candidatura o contratti frozen; nessun test/build, staging, commit, push, reset, checkout o pulizia. `.idea/**` non è stata esplorata.

La review termina con questo rapporto. L'implementazione rimane un incarico successivo.
