# Dossier tecnico del modello italiano Synthetic World

## A. Sintesi tecnica

### Stato verificato e perimetro

Il repository è sul branch `synthetic-world`, con HEAD:

```text
50ed99476cf22a8137d1adcf8e2c82f7235e0ec6
```

Branch e HEAD coincidono con quelli indicati. Non risultano modifiche tracked né staged; sono presenti numerosi artefatti untracked preesistenti, inclusi i due documenti di orientamento e gli artefatti D2-B2 descritti sotto.

Ho letto gli `AGENTS.md` applicabili, la guida e la ricognizione. L’attività è rimasta in sola lettura: nessun file creato o modificato, nessun generatore, build o test eseguito, nessun download, nessuna operazione Git di scrittura. `.idea/**` non è stata esplorata.

### Distinzione delle evidenze

Nel dossier:

- **Dati:** contenuto effettivamente ispezionato nei file locali.
- **Codice:** comportamento ricostruito dall’implementazione, senza eseguirla.
- **Contratto:** significato stabilito dai documenti normativi frozen.
- **Documentazione:** attestazioni storiche di produzione, review o provenienza.
- **Inferenza:** conseguenza tecnica delle evidenze, esplicitamente qualificata.
- **Non verificato:** informazione non dimostrata in questa ricognizione.

I conteggi e i campioni numerici riportati provengono dai file reali, non dalla fixture socioeconomica.

### Risultato principale

Il modello italiano attuale comprende tre componenti con rappresentazioni diverse:

| Componente | Rappresentazione attuale | Semantica esterna ai dati |
|---|---|---|
| Dati personali di base | Sei CSV embedded, identificati dai rispettivi SHA-256 | Coorti, intervalli temporali, composizione degli indirizzi, selezione geografica e altre regole Go |
| Modello socioeconomico | JSON numerico posizionale `SW-L2-D2B2-logical-v0` | Tassonomie, ordinali, assi, unità economiche e protocollo deterministico |
| Fotografie | Catalogo esterno minimale; separatamente, catalogo locale fakefacegen più ricco | Normalizzazione, scelta per genere/età, rendition e costruzione dei riferimenti |

**Il modello logico canonico è disponibile localmente:** [D2B2-logical-assets-v0.json](/Users/marco/src/krenalis/D2B2-logical-assets-v0.json:1). Il suo SHA-256 coincide con quello attestato dal production report e dalla review canonica. Non è quindi necessario sostituirlo con una fixture per studiarne i valori.

La difficoltà principale per una futura rappresentazione internazionale è che **molti numeri del JSON hanno significato soltanto insieme agli ordinali e alle formule Go**. Anche i CSV contengono etichette il cui significato operativo — soprattutto le coorti — dipende dal codice.

Questo dossier descrive tali dipendenze; non sceglie un nuovo formato.

---

## B. Dizionario completo dei sei CSV

### B.1 Regole comuni e cardinalità

Percorso comune:

```text
tools/fakedata/data/it-2026a/
```

| File | Record dati | Righe fisiche, intestazione inclusa | Intestazione originale |
|---|---:|---:|---|
| `age_weights.csv` | 55 | 56 | `age,weight` |
| `first_names.csv` | 800 | 801 | `gender,cohort,name,ascii_atom,weight` |
| `last_names.csv` | 600 | 601 | `name,ascii_atom,weight` |
| `places.csv` | 7.896 | 7.897 | `istat_code,city,state_prov,region,population` |
| `postal_codes.csv` | 7.896 | 7.897 | `istat_code,postal_codes` |
| `street_names.csv` | 400 | 401 | `name,weight` |

Il caricatore verifica anzitutto lo SHA-256 esatto dei byte. Richiede intestazioni esatte, assenza di BOM e CR, terminazione con LF; usa poi il parser CSV. Di conseguenza, nell’implementazione corrente anche modifiche puramente fisiche al CSV sono rilevanti per l’accettazione del file.

Le strutture runtime sono `italyData`, `ageRow`, `firstNameRow`, `nameRow`, `placeRow` e `firstNameGroup`. Pesi e popolazioni sono conservati come `uint64`; codici e testi restano stringhe. [Caricamento e tipi, righe 21–98 e 210–328](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:21).

**Distinzione importante:** alcune proprietà sono controllate semanticamente dal parser; altre sono fissate dal contratto, dai test o indirettamente dall’hash esatto. Non sono la stessa forma di validazione.

### B.2 `age_weights.csv`

[File originale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/age_weights.csv:1).

| Colonna | Rappresentazione | Tipo runtime | Significato e vincoli |
|---|---|---|---|
| `age` | Intero decimale testuale | `int` | Età compiuta alla data di ancoraggio; esattamente 21–75 |
| `weight` | Intero decimale positivo | `uint64` | Massa relativa di selezione dell’età |

Esempi originali:

```csv
age,weight
21,609156
75,630573
```

**Dati verificati**

- 55 età consecutive, una riga per età.
- Somma dei pesi: **41.485.820**.
- Nessuna distribuzione distinta per genere.
- I pesi rappresentano conteggi della popolazione residente delle età considerate, sommati sui comuni, secondo la provenienza POSAS documentata.

**Codice**

Il parser richiede esattamente 55 righe e verifica `age == posizione + 21`; ogni peso deve essere positivo. Il generatore seleziona l’età con probabilità `peso / somma_pesi`. [Parser](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:133), [selezione](/Users/marco/src/krenalis/tools/fakedata/layer1.go:359).

L’età selezionata è l’**AnchorAge**, riferita al **1° gennaio 2026**. Per un’età `a`, il giorno di nascita è selezionato uniformemente nell’intervallo inclusivo:

```text
inizio = anchor.AddDate(-a-1, 0, 1)
fine   = anchor.AddDate(-a,   0, 0)
```

L’insieme complessivo delle date producibili va quindi dal **1950-01-02 al 2005-01-01**.

`CurrentAge` è successivamente calcolata rispetto a `ReferenceDate`, ammessa tra `2026-01-01` e `2030-12-31`. Per una nascita il 29 febbraio, negli anni non bisestili l’anniversario è il 28 febbraio. La data di riferimento non cambia la nascita già determinata. [Regole temporali](/Users/marco/src/krenalis/tools/fakedata/layer1_helpers.go:26).

**Da distinguere**

- I 55 conteggi sono dati italiani.
- Intervallo 21–75, ancoraggio e formula di nascita sono anche vincoli dell’algoritmo corrente.
- Non sono proprietà necessarie di qualsiasi popolazione nazionale.

### B.3 `first_names.csv`

[File originale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/first_names.csv:1).

| Colonna | Rappresentazione | Tipo runtime | Significato |
|---|---|---|---|
| `gender` | `female` oppure `male` | `Gender` nel raggruppamento | Seleziona una delle due distribuzioni nominali |
| `cohort` | Una delle cinque stringhe sotto | `string` | Etichetta della distribuzione per anno di nascita |
| `name` | Testo Unicode | `display string` | Nome restituito nella persona |
| `ascii_atom` | Testo ASCII minuscolo | `asciiAtom string` | Atomo già preparato per indirizzi email |
| `weight` | `1`, `2` oppure `4` | `uint64` | Peso editoriale relativo nel gruppo |

Esempi originali:

```csv
gender,cohort,name,ascii_atom,weight
female,1950-1964,Ada,ada,2
male,1980-1989,Noè,noe,1
male,2000-2005,Vito,vito,1
```

#### Coorti effettive

| Etichetta originale | Anni di nascita inclusi dal Go | Nomi female | Nomi male |
|---|---|---:|---:|
| `1950-1964` | 1950–1964 | 80 | 80 |
| `1965-1979` | 1965–1979 | 80 | 80 |
| `1980-1989` | 1980–1989 | 80 | 80 |
| `1990-1999` | 1990–1999 | 80 | 80 |
| `2000-2005` | 2000–2005 | 80 | 80 |

Sono **cinque coorti e dieci gruppi**, non cinque suddivisioni ottenute dividendo l’intervallo delle età.

I confini sono codificati esplicitamente in `cohortForYear`, mediante confronti sull’anno della nascita effettivamente generata. Il CSV contiene soltanto l’etichetta: il parser non interpreta sintatticamente `1950-1964` come intervallo. [Confini nel Go](/Users/marco/src/krenalis/tools/fakedata/layer1_helpers.go:39).

Poiché la nascita massima è `2005-01-01`, il 2005 è effettivamente raggiungibile soltanto con quel giorno nel modello corrente.

#### Cardinalità, pesi e ordine

Ogni gruppo contiene:

- 20 nomi con peso 1;
- 40 nomi con peso 2;
- 20 nomi con peso 4;
- somma dei pesi **180**.

Il file contiene 314 stringhe `name` distinte globalmente; la ripetizione dello stesso nome in gruppi diversi è prevista. Le 800 triple `(gender, cohort, name)` sono distinte.

Il parser verifica valori ammessi, cardinalità 800, dieci gruppi completi, 80 righe per gruppo e quote dei pesi. Non esegue una verifica separata di ordinamento o duplicazione dei nomi; l’hash fissa comunque i byte attuali. [Parser dei nomi](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:163).

L’ordine contrattuale è:

1. genere, `female` prima di `male`;
2. coorte cronologica;
3. chiave Unicode `D(name)`;
4. nome originale come discriminante finale.

La rettifica definisce `D` mediante normalizzazione e full case folding Unicode 16. L’ordine non è decrescente per peso. Il runtime conserva l’ordine relativo delle righe nei gruppi e lo usa nella selezione cumulativa. [Rettifica, identità e ordinamento](/Users/marco/src/krenalis/L1-D1-B_Contract_Rectification.md:477).

#### Consumo e fallback

Il generatore:

1. seleziona `Gender` con probabilità uguale tra i due valori;
2. genera la nascita;
3. ricava la coorte dall’anno;
4. seleziona il gruppo `(gender, cohort)`;
5. effettua una selezione pesata tra gli 80 elementi.

Un anno fuori intervallo o un gruppo mancante produce errore: non è previsto un gruppo di fallback. `name` alimenta `Person.FirstName`; `ascii_atom` alimenta l’email. [Consumo dei nomi e genere](/Users/marco/src/krenalis/tools/fakedata/layer1.go:417).

Gli atomi ASCII sono dati già prodotti offline. Il parser verifica soltanto la loro forma — lettere `a`–`z` e trattino, senza trattino iniziale/finale — e non ricalcola la traslitterazione.

**Provenienza:** nomi curati con evidenze storiche e statistiche, non una tabella completa di frequenze nazionali. La rettifica per il gruppo più giovane usa evidenze annuali di posizione nelle prime dieci, non conteggi esaustivi. I pesi 1/2/4 sono classi editoriali di plausibilità. [Contratto D1-B](/Users/marco/src/krenalis/L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md:1), [rettifica](/Users/marco/src/krenalis/L1-D1-B_Contract_Rectification.md:1).

### B.4 `last_names.csv`

[File originale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/last_names.csv:1).

| Colonna | Rappresentazione | Tipo runtime | Significato |
|---|---|---|---|
| `name` | Testo Unicode | `display string` | Cognome restituito |
| `ascii_atom` | Atomo ASCII minuscolo | `asciiAtom string` | Componente dell’email |
| `weight` | `1`, `2`, `4` | `uint64` | Massa relativa editoriale |

Esempi originali:

```csv
name,ascii_atom,weight
Abate,abate,2
Saccà,sacca,1
Zorzi,zorzi,2
```

**Dati:** 600 cognomi distinti; 240 con peso 1, 240 con peso 2, 120 con peso 4; somma **1.200**.

**Codice:** esattamente 600 righe, validazione display/atomo, quote dei pesi. La distribuzione è nazionale unica: non è condizionata da comune, regione, genere o coorte. [Parser condiviso](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:331), [consumatore](/Users/marco/src/krenalis/tools/fakedata/layer1.go:453).

**Contratto:** ordine `D(name)`, poi nome originale; display Unicode e atomo ASCII sono valori distinti da conservare. Le collisioni tra atomi non costituiscono identità del cognome e sono consentite.

L’email può usare nome, iniziale e cognome ASCII secondo quattro schemi pesati:

| Schema | Peso |
|---|---:|
| `first.last.token` | 40 |
| `initial.last.token` | 25 |
| `first.token` | 20 |
| `first-last.token` | 15 |

Il token deriva da un meccanismo deterministico separato; non è il cognome a garantire l’unicità. [Costanti e costruzione email](/Users/marco/src/krenalis/tools/fakedata/layer1.go:62).

### B.5 `places.csv`

[File originale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/places.csv:1).

| Colonna | Rappresentazione | Tipo runtime | Significato e vincoli |
|---|---|---|---|
| `istat_code` | Stringa di sei cifre ASCII | `istatCode string` | Codice comunale ISTAT nella fotografia territoriale adottata |
| `city` | Testo Unicode | `municipality string` | Denominazione comunale |
| `state_prov` | Due lettere ASCII maiuscole | `stateProv string` | Sigla dell’entità provinciale/UTS della fonte |
| `region` | Denominazione Unicode | `region string` | Regione italiana; raccordo con l’asse socioeconomico |
| `population` | Intero decimale positivo | `population uint64` | Popolazione comunale, tutte le età |

Esempi originali:

```csv
istat_code,city,state_prov,region,population
001001,Agliè,TO,Piemonte,2584
119024,Teulada,SU,Sardegna,3156
```

**Dati verificati**

- 7.896 comuni.
- 20 valori regionali.
- 110 valori `state_prov`.
- Popolazione totale: **58.942.828**.
- Minimo comunale: **30**; massimo: **2.745.062**.
- Ordinamento strettamente crescente per `istat_code`.

Il codice ISTAT è una **stringa identificativa**: `001001` non può essere conservato senza perdita trasformandolo nel numero `1001`.

Il parser richiede sei cifre, unicità e ordine crescente; denominazioni UTF-8 non vuote e prive di spazi esterni; sigla di due maiuscole; popolazione positiva. Non verifica qui che `region` appartenga all’elenco socioeconomico. [Parser](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:369).

#### Selezione geografica

La scelta del comune è una miscela:

- **95%**: distribuzione proporzionale alla popolazione;
- **5%**: distribuzione uniforme sui comuni.

Nel ramo pesato il runtime divide i comuni, nell’ordine del file, in bucket contigui da 128: **62 bucket**, dei quali 61 completi e l’ultimo da 88 elementi. Seleziona prima un bucket secondo la somma delle popolazioni, poi un comune all’interno secondo la sua popolazione.

Il risultato resta proporzionale alla popolazione nel ramo pesato; la suddivisione in bucket è un dettaglio dell’algoritmo con effetti sul consumo dello stream. [Bucket](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:100), [selezione](/Users/marco/src/krenalis/tools/fakedata/layer1.go:347).

#### Geografia e modello socioeconomico

La persona espone `Municipality`, `StateProv`, `PostalCode` e `Country`; non espone direttamente il codice ISTAT e la regione dentro `Address`.

Il modello socioeconomico ricalcola lo **stesso comune** con la medesima coordinata della persona, legge `place.region` e cerca una corrispondenza testuale esatta nell’array regionale Go. Una stringa non riconosciuta produce errore. Non viene ricavata la regione dalla sigla provinciale o dalle cifre ISTAT. [Raccordo geografico](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_facts.go:55).

Il CSV rappresenta l’associazione:

```text
comune → sigla provinciale/UTS
comune → denominazione regionale
comune → popolazione
```

Non contiene tabelle autonome di province e regioni, codici regionali, geometrie, gerarchie amministrative generiche o un collegamento a un’entità territoriale internazionale.

**Provenienza:** SITUAS e POSAS riferiti al 1° gennaio 2026; sono documentati un raccordo per ricodifiche sarde e normalizzazioni delle denominazioni. `state_prov` è la sigla della fotografia territoriale usata, non una garanzia generale sulla validità corrente come sigla automobilistica. [Fonti e produzione](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/SOURCES.md:1).

### B.6 `postal_codes.csv`

[File originale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/postal_codes.csv:1).

| Colonna | Rappresentazione | Tipo runtime | Significato |
|---|---|---|---|
| `istat_code` | Sei cifre ASCII | Chiave `string` | Collegamento a un comune di `places.csv` |
| `postal_codes` | Uno o più CAP separati da `;` | `[]string` | Insieme ordinato di CAP utilizzabili per quel comune |

Esempi originali:

```csv
istat_code,postal_codes
001001,10011
001272,10100;10125;10130;10135;10140;10145;10150;10156
119024,09019
```

Distribuzione effettiva del numero di CAP per comune:

| CAP nella riga | Comuni |
|---:|---:|
| 1 | 7.841 |
| 2 | 20 |
| 3 | 3 |
| 4 | 3 |
| 5 | 2 |
| 7 | 1 |
| 8 | 26 |

**Parser:** una riga per ogni comune, nessun codice comunale sconosciuto o duplicato, copertura completa; CAP non vuoti, ciascuno di cinque cifre ASCII, ordinati strettamente e senza duplicati nella stessa riga. [Parser postale](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:406).

Il parser non impone separatamente il limite di otto CAP o l’ordine delle righe. Queste proprietà sono presenti nel dataset, fissate dall’hash e controllate dai test pertinenti. [Test di copertura e ordine](/Users/marco/src/krenalis/tools/fakedata/italy_data_test.go:117).

**Selezione:** uniforme tra i CAP del comune scelto, senza pesi propri. L’ordine della lista determina l’associazione tra indice casuale e CAP. Un CAP non è globalmente univoco tra comuni.

**Provenienza:** GeoNames, integrazioni e override documentati mediante IPA. La documentazione distingue esplicitamente questa copertura operativa da un inventario postale esaustivo e autoritativo. La riduzione a un massimo operativo di otto valori è parte della produzione italiana attuale.

Il legame comune–codici postali è un’informazione distinta dalla sintassi italiana di cinque cifre: entrambe vanno conservate, senza confonderle.

### B.7 `street_names.csv`

[File originale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/street_names.csv:1).

| Colonna | Rappresentazione | Tipo runtime | Significato |
|---|---|---|---|
| `name` | Testo Unicode | `nameRow.display` | Frammento toponomastico da inserire nell’indirizzo |
| `weight` | `1`, `2`, `4` | `uint64` | Massa relativa editoriale |

Esempi originali:

```csv
name,weight
Abruzzo,1
dell'Unità,4
XXVII Gennaio,2
```

**Dati:** 400 frammenti distinti; 160 pesi 1, 160 pesi 2, 80 pesi 4; somma **800**. Ordine contrattuale per `D(name)` e nome originale, non per frequenza.

Il campo `name` non contiene necessariamente una strada completa e non rappresenta un elenco di strade realmente associate a ciascun comune. Il lessico è condiviso dalla generazione italiana.

#### Composizione dell’indirizzo

| Parte | Origine | Regola |
|---|---|---|
| Tipo di strada | Costanti Go | `Via`, `Viale`, `Piazza`, `Corso`, `Largo`, `Vicolo` |
| Pesi del tipo | Costanti Go | Rispettivamente 55, 15, 12, 8, 5, 5 |
| Nome/frammento | CSV | Selezione secondo `weight` |
| Numero civico | Algoritmo Go | `1 + min(U₁, U₂)`, con entrambi uniformi in 0–249 |
| Presenza suffisso | Algoritmo Go | Probabilità 7/100 |
| Suffisso | Costanti Go | Uniforme tra `/A`, `/B`, `/C`, `bis` |
| Spaziatura | Go | `tipo + spazio + frammento + spazio + civico`; `bis` preceduto da spazio |
| Paese | Costante Go | `IT` |

Il civico è quindi compreso tra 1 e 250, con preferenza per i numeri più bassi. [Costanti](/Users/marco/src/krenalis/tools/fakedata/layer1.go:62), [composizione](/Users/marco/src/krenalis/tools/fakedata/layer1.go:265).

### B.8 Provenienza e differenza tra contratto e parser

Per i dataset D1-B, il contratto stabilisce una forma Unicode e tipografica più restrittiva della sola validazione runtime: NFC, case folding Unicode 16 per l’ordine, regole su spazi, apostrofi, trattini e atomi ASCII. Il parser attuale non ricostruisce l’intero processo di normalizzazione: accetta i file frozen identificati dai loro hash. [Regole testuali del contratto](/Users/marco/src/krenalis/L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md:665).

La provenienza distingue:

- **Età e popolazioni:** aggregazioni di dati demografici.
- **Geografia:** fotografia territoriale e raccordi.
- **CAP:** copertura operativa da fonti e override.
- **Nomi:** curazione informata da evidenze.
- **Cognomi e strade:** lessici curati, con pesi editoriali.

Non è corretto attribuire ai pesi 1/2/4 lo stesso significato statistico dei conteggi di popolazione.

Nelle evidenze postali locali coesistono attestazioni `PASS` e documenti ancora `PROPOSED`/`PENDING`; inoltre alcuni campi di review non sono compilati. Questa è una lacuna della catena documentale, non un’assenza dei valori CSV ispezionati. [Revisione postale locale](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/provenance/postal-override-review.md:1).

---

## C. Dizionario strutturale del modello socioeconomico

### C.1 Disponibilità e identità del modello canonico

| Proprietà | Valore verificato |
|---|---|
| File logico locale | `D2B2-logical-assets-v0.json` |
| Dimensione | 685.312 byte |
| SHA-256 | `f049103241e6e43ffb5dd4a1247b64b2d0ca1bff787950f25284b5b435fd21a8` |
| `/schema` | `SW-L2-D2B2-logical-v0` |
| Percorso richiesto dal runtime | `synthetic-world/it-2026a/model.json` |
| `/companyCount` | 65.536 |

L’hash locale coincide con le attestazioni del [production report](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PRODUCTION_REPORT.md:1) e della [review canonica](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_ASTRA_CANONICAL_ASSET_REVIEW.md:1).

La corrispondenza con l’asset effettivamente pubblicato all’URL runtime **non è stata verificata**, perché non sono stati scaricati asset esterni.

`core/testdata/socioeconomic-model.json` è una fixture artificiale distinta e non è stata usata per sostituire i valori canonici.

### C.2 Albero completo del JSON

Questo è l’albero delle proprietà esistenti, non una proposta di formato:

```text
/
├── schema
├── companyCount
├── ages
│   ├── Education
│   │   ├── lowerBounds
│   │   └── compiled
│   ├── PrimaryActivity
│   │   ├── lowerBounds
│   │   └── compiled
│   ├── Employee.JobLevel
│   │   ├── lowerBounds
│   │   └── compiled
│   ├── Income[s]
│   │   ├── lowerBounds
│   │   └── compiled
│   ├── Income[u]
│   │   ├── lowerBounds
│   │   └── compiled
│   ├── Income[t]
│   │   ├── lowerBounds
│   │   └── compiled
│   ├── Income[r]
│   │   ├── lowerBounds
│   │   └── compiled
│   └── Income[i]
│       ├── lowerBounds
│       └── compiled
├── companyIndustry
│   └── weights
├── employeeCount
│   └── laws[]
│       ├── values
│       └── weights
├── education
│   └── weights
├── primaryActivity
│   └── weights
├── employeeJobFamily
│   └── weights
├── employeeJobLevel
│   └── weights
├── selfEmployedJobFamily
│   └── weights
├── selfEmployedIndustry
│   └── weights
├── employer
│   ├── cap
│   ├── industryFactors
│   └── sizeFactors
└── income
    ├── e
    ├── s
    ├── u
    ├── t
    ├── r
    └── i
        [ciascun ramo contiene]
        ├── tupleMap
        └── laws[]
            ├── values
            └── weights
```

Sono tredici proprietà principali. Gli oggetti sono chiusi: proprietà sconosciute o mancanti vengono rifiutate. [Tipi JSON e decoder](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_json.go:16).

### C.3 Tipi, dimensioni e consumatori

Tutti i numeri del modello logico sono **interi JSON non negativi**. Il decoder rifiuta numeri frazionari, notazione esponenziale e valori negativi; non li converte tramite floating point.

Nella tabella, le dimensioni indicate come “effettive” sono quelle del modello canonico locale.

| JSON Pointer | Tipo JSON → Go | Cardinalità e significato |
|---|---|---|
| `/schema` | stringa → controllo letterale | Esattamente `SW-L2-D2B2-logical-v0` |
| `/companyCount` | numero → `uint64` | 1–104.855; effettivo 65.536; numero di Company operative |
| `/ages/<consumatore>/lowerBounds` | array di numeri → `[]uint32` | Limiti inferiori dei bucket d’età; primo 21, strettamente crescenti, massimo 75 |
| `/ages/<consumatore>/compiled` | array di numeri → `[]uint32`, poi `[55]uint8` | 55 ordinali: posizione `a-21` per ogni età 21–75 |
| `/companyIndustry/weights` | array → `[]uint32` | 14 masse, una per Industry |
| `/employeeCount/laws` | array di oggetti | Esattamente 14 leggi, indicizzate da Industry |
| `/employeeCount/laws/k/values` | array → `[]uint32` | Supporti del numero di dipendenti; 1–128 punti per legge |
| `/employeeCount/laws/k/weights` | array → `[]uint32` | Masse positive associate uno a uno ai supporti |
| `/education/weights` | array → `[]uint32` | Effettivi 840 = 6×20×7 |
| `/primaryActivity/weights` | array → `[]uint32` | Effettivi 6.720 = 8×7×20×6 |
| `/employeeJobFamily/weights` | array → `[]uint32` | 1.400 = 7×20×10 |
| `/employeeJobLevel/weights` | array → `[]uint32` | Effettivi 1.680 = 6×7×10×4 |
| `/selfEmployedJobFamily/weights` | array → `[]uint32` | 1.400 = 7×20×10 |
| `/selfEmployedIndustry/weights` | array → `[]uint32` | 140 = 10×14 |
| `/employer/cap` | numero → `uint16` | 1–65.535; effettivo 4.096 |
| `/employer/industryFactors` | array → `[140]uint16` | 10×14 fattori; ciascuno 1–4.095 |
| `/employer/sizeFactors` | array → `[16]uint16` | 4×4 fattori; ciascuno 1–4.095 |
| `/income/<ramo>/tupleMap` | array → `[]uint32`, poi `[]uint16` | Associa ciascuna tupla a una legge del ramo |
| `/income/<ramo>/laws/k/values` | array → `[]uint32` | Supporti monetari, da 0 a 4.294.967.295 EUR interi |
| `/income/<ramo>/laws/k/weights` | array → `[]uint32` | Masse positive, allineate ai supporti |

Per le distribuzioni categoriche il runtime conserva `socioeconomicWeights`: larghezza della riga e somme cumulative `uint64`. Per le leggi discrete conserva `socioeconomicLaw`: supporti `uint32` e cumulative `uint64`. I pesi originali restano recuperabili come differenze delle cumulative. [Modello runtime](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model.go:25).

### C.4 Partizioni d’età effettive

Le otto partizioni sono proprietà distinte, anche quando contengono gli stessi valori.

| Proprietà sotto `/ages` | `lowerBounds` effettivo | Intervalli inclusivi |
|---|---|---|
| `Education` | `[21,25,35,45,55,65]` | 21–24, 25–34, 35–44, 45–54, 55–64, 65–75 |
| `PrimaryActivity` | `[21,25,35,45,55,60,65,70]` | 21–24, 25–34, 35–44, 45–54, 55–59, 60–64, 65–69, 70–75 |
| `Employee.JobLevel` | `[21,25,35,45,55,65]` | Come `Education` |
| `Income[s]` | `[21,25,35,45,55,65]` | Come `Education` |
| `Income[u]` | `[21,25,35,45,55,65]` | Come `Education` |
| `Income[t]` | `[21,25,30,40,55,65]` | 21–24, 25–29, 30–39, 40–54, 55–64, 65–75 |
| `Income[r]` | `[21,45,55,60,65,70]` | 21–44, 45–54, 55–59, 60–64, 65–69, 70–75 |
| `Income[i]` | `[21,25,35,45,55,65]` | Come `Education` |

Il decoder ammette fino a sei bucket per ogni consumatore, salvo `PrimaryActivity`, che ne ammette otto. Non esiste `Income[e]` tra le partizioni: il reddito employee non ha un asse d’età diretto.

`compiled` deve coincidere esattamente con la partizione descritta da `lowerBounds`; non è una seconda classificazione indipendente. [Validazione delle età](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_compile.go:78).

### C.5 Validazioni numeriche e strutturali

| Struttura | Regole verificate dal compilatore |
|---|---|
| Distribuzione categorica | Dimensione completa; pesi anche zero ammessi; somma di ogni riga positiva; MCD dei pesi della riga uguale a 1 |
| Legge discreta | Supporti e pesi di uguale lunghezza positiva; supporti strettamente crescenti; pesi tutti positivi; MCD 1 |
| `employeeCount` | 14 leggi; deve esistere supporto di forza lavoro positivo in almeno un’Industry con massa positiva |
| Fattori Employer | Cardinalità esatta, valori positivi nei limiti |
| Leggi Income | Nessun duplicato canonico nello stesso ramo; tutte referenziate |
| `tupleMap` | Copertura completa; riferimenti validi; ID delle leggi attribuiti nell’ordine del primo incontro |
| Employee, fascia `zero` | Ogni quinta posizione della mappa deve contenere il sentinel 65.535 |
| Documento | Un solo valore JSON, limite di 8 MiB; limiti preventivi su vettori e token |

Il compilatore **verifica** la forma canonica; non ordina supporti, non fonde duplicati e non divide automaticamente i pesi per il MCD. Tali operazioni appartengono alla produzione precedente alla pubblicazione. [Compilazione](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_compile.go:12), [limiti e decoder](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_json.go:162).

Nel modello effettivo tutti i 12.194 pesi categorici sono positivi. Il totale di una riga non è universalmente `2³²−1`: alcune righe canoniche hanno totali inferiori dopo riduzione per MCD. Non bisogna quindi sostituire i pesi con una convenzione di totale fisso.

### C.6 Produzione offline e artefatti di evidenza

Oltre al modello logico sono disponibili:

| File locale | Ruolo | SHA-256 verificato |
|---|---|---|
| `D2B2-publication-v3.json` | Manifesto ed evidenze di pubblicazione | `c2ee58c9b1cc59e2388e1f78036379dda9d519b14d696425348000a31f24ae4d` |
| `D2B2-crosswalks-v3.json` | Raccordi dalle classificazioni sorgente | `e9bf549895d61d819996ab82415c6765d91ea5bfff7e4a391bcc4d9eaa2ca1a3` |
| `D2B2-producer-v3.py` | Producer/strumento di qualificazione locale | `6716ee5e285236cac8169a1aef3968f5173ce82259eab315bb81af7837f4d7df` |

Il manifesto ha schema `SW-L2-D2B2-PUBLICATION-v3` e contiene riferimenti agli artefatti, audit logici, fonti, configurazione, provenienza, riproducibilità e risultati di qualificazione. **Non è il documento caricato dal runtime.**

I sei crosswalk dichiarati sono:

| Identificatore | Elementi |
|---|---:|
| `SW-ACT-LFS24-v0` | 593 |
| `SW-EDU-QUAL24-v0` | 50 |
| `SW-IND-ATECO22-v0` | 43 |
| `SW-JF-ISCO08-MAJOR-v0` | 43 |
| `SW-JL-RESP-v0` | 7 |
| `SW-REG-IT-v0` | 86 |

I raccordi possono contenere coefficienti razionali esatti con numeratore e denominatore testuali; non sono probabilità floating point da approssimare. [Crosswalk](/Users/marco/src/krenalis/D2B2-crosswalks-v3.json:1).

#### Fonti e passaggi ricostruibili

La pubblicazione identifica 19 input comprendenti:

- archivi LFS/ASIA, ASIA/FRAME e istruzione;
- SHIW, linee guida SILC;
- tre snapshot Eurostat;
- documenti INPS su pensioni e salari, SES;
- classificazioni ISCO/CP, ATECO e raccordi;
- relativi documenti metodologici.

ISCED è inoltre presente come riferimento metodologico. Gli input hanno ruoli diversi: calibrazione, controllo di componenti, crosswalk o evidenza metodologica. Non tutto ciò che compare nella pubblicazione è un dato numerico direttamente consumato dal modello.

La provenienza classifica i 24 componenti logici come `MODELED-ASSUMPTION`: il modello è informato da dati, ma non è una trascrizione integrale di statistiche ufficiali. [Manifesto di pubblicazione](/Users/marco/src/krenalis/D2B2-publication-v3.json:1).

#### Discrepanza documentale confermata

Il production report dichiara `CANONICAL PRODUCTION`, `SUCCESS` e produzione con 19 input fresh. Il producer locale, pur avendo l’hash citato dal report, si presenta nell’intestazione e nell’interfaccia come strumento **non canonico di qualificazione**. [Intestazione producer](/Users/marco/src/krenalis/D2B2-producer-v3.py:1), [report](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PRODUCTION_REPORT.md:1).

Questo consente di affermare:

- l’identità dei byte logici locali è attestata e verificabile;
- esiste una discrepanza sulla qualificazione e sull’uso del producer;
- la riproducibilità completa della produzione canonica non è stata nuovamente dimostrata;
- la discrepanza non autorizza a sostituire o rigenerare il modello frozen.

Il runtime acquisisce solo il modello logico tramite External Assets; non esegue il producer e non interpreta publication/crosswalk. [Amendment normativo](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md:116).

---

## D. Tassonomie, assi e dipendenze

### D.1 Dove sono definite le classificazioni

Le stringhe pubbliche sono definite in [socioeconomic_types.go](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_types.go:7). Gli ordini ordinali sono dichiarati esplicitamente in [socioeconomic_facts.go, righe 12–45](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_facts.go:12).

Il JSON logico **non contiene questi elenchi**. Un vettore di 14 numeri non descrive da solo quale numero rappresenti `agriculture`.

Gli ordinali seguenti sono tutti a base zero. Per classificazioni nominali l’ordinale è una posizione di protocollo, non un rango economico.

### D.2 Regioni: elenco completo

| Ordinale | Stringa esatta | Ordinale | Stringa esatta |
|---:|---|---:|---|
| 0 | `Abruzzo` | 10 | `Molise` |
| 1 | `Basilicata` | 11 | `Piemonte` |
| 2 | `Calabria` | 12 | `Puglia` |
| 3 | `Campania` | 13 | `Sardegna` |
| 4 | `Emilia-Romagna` | 14 | `Sicilia` |
| 5 | `Friuli-Venezia Giulia` | 15 | `Toscana` |
| 6 | `Lazio` | 16 | `Trentino-Alto Adige/Südtirol` |
| 7 | `Liguria` | 17 | `Umbria` |
| 8 | `Lombardia` | 18 | `Valle d'Aosta/Vallée d'Aoste` |
| 9 | `Marche` | 19 | `Veneto` |

La relazione con `places.csv` è per uguaglianza della stringa regionale, comprese grafia, apostrofi e Unicode.

### D.3 Education

Education significa **massimo livello di istruzione completato alla data di ancoraggio**.

| Ordinale | Valore serializzato | Significato |
|---:|---|---|
| 0 | `below-primary` | Istruzione primaria non completata |
| 1 | `primary` | Primaria completata |
| 2 | `lower-secondary` | Secondaria inferiore completata |
| 3 | `upper-secondary` | Secondaria superiore completata |
| 4 | `post-secondary-non-tertiary` | Post-secondaria non terziaria |
| 5 | `tertiary` | Terziaria, escluso il dottorato di ricerca |
| 6 | `doctoral` | Dottorato di ricerca |

`below-primary` è una categoria reale, non “dato mancante”. `tertiary` comprende i livelli terziari previsti dal contratto; `doctoral` non indica genericamente il titolo professionale di “dottore”.

Il contratto non autorizza a trattare questi sette valori come identici, senza raccordo, a qualunque classificazione ufficiale nazionale. [Contract 3, istruzione](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:133).

### D.4 PrimaryActivity

| Ordinale | Valore | Ramo Income | Significato |
|---:|---|---|---|
| 0 | `employee` | `e` | Attività principale da dipendente |
| 1 | `self-employed` | `s` | Attività principale autonoma |
| 2 | `unemployed` | `u` | Disoccupato |
| 3 | `student` | `t` | Studente |
| 4 | `retired` | `r` | Pensionato/ritirato secondo la classificazione del modello |
| 5 | `inactive` | `i` | Altra inattività |

È una classificazione unica dell’attività **principale**, non un’affermazione di assenza di attività secondarie o di altre fonti di reddito. La selezione deriva dalla distribuzione, senza introdurre nel runtime una gerarchia deterministica di precedenza tra ruoli.

### D.5 Industry

| Ordinale | Valore | Ambito contrattuale sintetico |
|---:|---|---|
| 0 | `agriculture` | Agricoltura, silvicoltura, pesca |
| 1 | `production-and-utilities` | Estrazione, manifattura, energia, acqua, rifiuti e attività correlate |
| 2 | `construction` | Costruzioni e installazioni fisiche connesse |
| 3 | `commerce` | Commercio di beni, incluso online |
| 4 | `transport-and-logistics` | Trasporto, consegne, magazzinaggio, logistica |
| 5 | `hospitality` | Alloggio e ristorazione |
| 6 | `information-and-communication` | Informatica, dati, telecomunicazioni, editoria |
| 7 | `finance-and-real-estate` | Finanza, assicurazioni, attività immobiliari |
| 8 | `professional-and-business-services` | Servizi professionali e di supporto alle imprese |
| 9 | `public-administration-and-safety` | Amministrazione, difesa, sicurezza ed emergenza secondo il contratto |
| 10 | `education` | Istruzione e formazione |
| 11 | `health-and-social-care` | Sanità, attività veterinarie, assistenza sociale |
| 12 | `arts-and-recreation` | Cultura, spettacolo, intrattenimento, sport |
| 13 | `other-services` | Altri servizi definiti, inclusi servizi personali e alcune riparazioni |

`other-services` non è un contenitore di valori sconosciuti. Queste sono categorie sintetiche del modello; non coincidono automaticamente con codici ATECO. [Contract 3, Industry](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:187).

### D.6 JobFamily

| Ordinale | Valore | Significato funzionale |
|---:|---|---|
| 0 | `production-and-fieldwork` | Produzione e lavoro operativo sul campo |
| 1 | `skilled-trades` | Mestieri qualificati, lavorazioni e riparazioni manuali |
| 2 | `transport-and-logistics` | Funzioni di trasporto e logistica |
| 3 | `customer-and-personal-service` | Servizio al cliente e alla persona |
| 4 | `administration` | Funzioni amministrative |
| 5 | `sales-and-commercial` | Vendite e funzioni commerciali |
| 6 | `technical-and-scientific` | Tecnica, ingegneria, informatica, laboratorio e ricerca |
| 7 | `professional-and-creative` | Professioni e attività creative secondo il contratto |
| 8 | `care-and-education` | Cura, insegnamento e formazione |
| 9 | `protective-services` | Funzioni di protezione e sicurezza |

È la funzione svolta dalla persona, distinta dal settore dell’azienda. Per esempio, ricerca scientifica sanitaria e cura clinica diretta appartengono a distinzioni funzionali diverse; il settore sanitario non determina da solo la JobFamily.

Tutte le combinazioni strutturali previste sono rappresentabili: il runtime non introduce un filtro generale Industry→JobFamily. [Contract 3, JobFamily](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:328).

### D.7 JobLevel e CompanySizeBand

| Ordinale | JobLevel | Significato |
|---:|---|---|
| 0 | `individual-contributor` | Responsabilità sul proprio lavoro |
| 1 | `supervisor` | Coordinamento operativo quotidiano |
| 2 | `manager` | Responsabilità su obiettivi e risorse di un’unità |
| 3 | `executive` | Direzione e governo strategico |

JobLevel misura responsabilità organizzativa, non anzianità o età. Si applica solo al ramo employee. [Contract 3, livelli](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:404).

| Ordinale | CompanySizeBand | EmployeeCount |
|---:|---|---:|
| 0 | `zero` | 0 |
| 1 | `micro` | 1–9 |
| 2 | `small` | 10–49 |
| 3 | `medium` | 50–249 |
| 4 | `large` | 250–4.294.967.295 |

La fascia è derivata dal numero di dipendenti. Non dipende dal fatturato e non viene campionata autonomamente. [Derivazione](/Users/marco/src/krenalis/tools/fakedata/socioeconomic.go:210).

### D.8 Assi e formule di indicizzazione

Notazione:

- `A`: bucket d’età **del consumatore interessato**;
- `E`: Education, 0–6;
- `R`: regione, 0–19;
- `F`: JobFamily, 0–9;
- `J`: JobLevel, 0–3;
- `I`: Industry, 0–13;
- `B`: CompanySizeBand, 0–4.

Gli array sono appiattiti con l’ultimo asse che varia più rapidamente.

| Struttura | Assi, nell’ordine | Forma effettiva | Indice o riga |
|---|---|---|---|
| Company Industry | `I` | 14 | `I` |
| Education | `A,R,E` | 6×20×7 | `(A*20+R)*7+E` |
| PrimaryActivity | `A,E,R,attività` | 8×7×20×6 | Riga `(A*7+E)*20+R`, larghezza 6 |
| Employee JobFamily | `E,R,F` | 7×20×10 | `(E*20+R)*10+F` |
| Employee JobLevel | `A,E,F,J` | 6×7×10×4 | `((A*7+E)*10+F)*4+J` |
| SelfEmployed JobFamily | `E,R,F` | 7×20×10 | Come Employee JobFamily |
| SelfEmployed Industry | `F,I` | 10×14 | `F*14+I` |
| EmployeeCount | `I` → legge | 14 leggi | `laws[I]` |
| Employer Industry factor | `F,I` | 10×14 | `F*14+I` |
| Employer Size factor | `J,B positivo` | 4×4 | `J*4+(B-1)` |
| Income employee | `R,F,J,I,B` | 20×10×4×14×5 | `(((R*10+F)*4+J)*14+I)*5+B` |
| Income self-employed | `A,R,F,I` | 6×20×10×14 | `((A*20+R)*10+F)*14+I` |
| Income u/t/r/i | `A,E,R` | 6×7×20 | `(A*7+E)*20+R` |

Per Income, l’indice seleziona una voce di `tupleMap`, che identifica una legge; non seleziona direttamente un importo. [Formule operative](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_facts.go:71).

#### Cardinalità Income

| Ramo | Slot `tupleMap` effettivi | Leggi effettive | Massimo leggi ammesso | Punti per legge effettivi / massimo |
|---|---:|---:|---:|---:|
| `e` | 56.000 | 16 | 512 | 385 / 512 |
| `s` | 16.800 | 12 | 256 | 385 / 512 |
| `u` | 840 | 8 | 64 | 385 / 512 |
| `t` | 840 | 8 | 64 | 385 / 512 |
| `r` | 840 | 8 | 64 | 385 / 512 |
| `i` | 840 | 8 | 64 | 385 / 512 |

Nel ramo employee, 11.200 slot corrispondono alla fascia `zero` e contengono 65.535. Gli slot raggiungibili sono 44.800. Il sentinel non è un ID di legge e non significa reddito zero.

### D.9 Esempi numerici canonici

Tutti gli esempi seguenti provengono dal [modello canonico locale](/Users/marco/src/krenalis/D2B2-logical-assets-v0.json:1).

**Education:** `/education/weights/0`–`6` è la distribuzione per età 21–24, regione Abruzzo:

```text
[2735501, 19122730, 524395639, 3161678165,
 97788024, 487775224, 1472012]
```

Le posizioni corrispondono, nell’ordine, ai sette livelli Education. Il valore `3161678165` è quindi la massa di `upper-secondary` per quella combinazione.

**PrimaryActivity:** i primi sei pesi, per età 21–24, `below-primary`, Abruzzo, sono:

```text
[999546793, 264282080, 433316994,
 1880866217, 2407818, 714547393]
```

**Income employee:**

- `/income/e/tupleMap/7 = 0`: Abruzzo, `production-and-fieldwork`, `individual-contributor`, `production-and-utilities`, fascia `small`.
- `/income/e/tupleMap/8 = 1`: stessa combinazione, fascia `medium`.

Primi punti delle leggi:

| Legge | Importo EUR | Peso |
|---|---:|---:|
| `e/laws/0` | 0 | 8.589.935 |
| `e/laws/0` | 382 | 11.162.442 |
| `e/laws/0` | 1.144 | 11.162.442 |
| `e/laws/1` | 0 | 8.589.935 |
| `e/laws/1` | 461 | 11.162.442 |
| `e/laws/1` | 1.379 | 11.162.442 |

**EmployeeCount, Industry agriculture:** primi punti della legge 0:

```text
valore 0 → peso 128849016
valore 1 → peso 2978218771
valore 2 → peso 720631680
```

Le 14 leggi effettive contengono ciascuna 128 supporti, da 0 a 131.072. Il limite runtime del tipo resta `uint32`: 131.072 è il massimo presente nell’asset, non il massimo rappresentabile.

### D.10 Dipendenze di persone e aziende

La sequenza verificata è:

```text
nascita → AnchorAge
comune → regione
AnchorAge + regione → Education
AnchorAge + Education + regione → PrimaryActivity
```

Poi:

| Ramo | Dipendenze successive |
|---|---|
| Employee | Education+regione → JobFamily; età+Education+JobFamily → JobLevel; JobFamily+JobLevel → Employer; Employer → Industry e SizeBand; questi valori più regione/famiglia/livello → Income |
| SelfEmployed | Education+regione → JobFamily; JobFamily → Industry; età+regione+famiglia+Industry → Income |
| Unemployed, student, retired, inactive | Partizione d’età propria+Education+regione → Income |

L’età usata è sempre quella di ancoraggio. Il genere non è un asse del modello socioeconomico attuale.

Le Company sono già entità operative del motore:

```text
Company {
    ID
    EmployeeCount
    Industry
    SizeBand
}
```

Sono generate per coordinate da 1 a `companyCount`, con Industry e EmployeeCount deterministici. La loro esistenza non dipende dall’esposizione come entità della Simulated Application. [Costruzione e accesso alle Company](/Users/marco/src/krenalis/tools/fakedata/socioeconomic.go:26).

Per ogni coppia `(JobFamily, JobLevel)` viene costruito un indice Employer. Il peso di una Company è:

```text
min(EmployeeCount, cap)
× industryFactors[F,I]
× sizeFactors[J,B-1]
```

Le aziende con zero dipendenti non partecipano. Gli indici sono 10×4 = **40**, con Company ordinate per coordinata. Il runtime verifica overflow e totale positivo.

`EmployeeCount` non è un contatore delle persone già assegnate. Le richieste non consumano posti di lavoro e non decrementano capacità. [Semantica contrattuale delle aziende](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:246).

Le strutture generate sono:

- `SocioeconomicFacts`: Education, PrimaryActivity, eventuale Employee, eventuale SelfEmployed, Income;
- `EmployeeFacts`: riferimento Employer, JobFamily, JobLevel;
- `SelfEmployedFacts`: Industry, JobFamily.

Industry e dimensione dell’Employer vengono ricavate dalla Company. Il ramo self-employed non riceve automaticamente un riferimento Company. [Tipi risultanti](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_types.go:103).

### D.11 Redditi: significato, precisione e campionamento

#### Significato comune ai sei rami

Income è un **aggregato monetario personale lordo annualizzato**, alla situazione di ancoraggio, espresso in **euro nominali interi**. Non è fatturato aziendale, reddito familiare, patrimonio o necessariamente il totale storico degli ultimi dodici mesi.

Comprende, secondo il contratto, flussi personali pertinenti come retribuzioni, utili personali positivi da lavoro autonomo al netto dei costi dell’attività, pensioni, benefici monetari, borse e rendite monetarie ammesse. Esclude trasferimenti patrimoniali, prestiti, rimborsi e altre componenti escluse dal contratto. Le perdite d’impresa non diventano reddito negativo né compensano altri flussi.

Tutti i rami rappresentano lo stesso concetto economico; cambiano i condizionamenti e le distribuzioni. Un non occupato può avere reddito positivo; un employee può avere zero. [Contract 5, definizione economica](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:67).

#### Valori e distribuzioni effettive

| Ramo | Massimo presente nelle sue leggi, EUR | Peso dello zero in ogni legge del ramo |
|---|---:|---:|
| Employee | 1.468.132 | 8.589.935 |
| Self-employed | 1.261.067 | 343.597.353 |
| Unemployed | 125.552 | 1.503.238.420 |
| Student | 163.928 | 2.791.728.493 |
| Retired | 196.770 | 42.949.670 |
| Inactive | 208.634 | 2.362.231.802 |

Nel modello canonico ogni legge Income ha 385 supporti, compreso zero, e somma dei pesi **4.294.967.295**.

Zero è un supporto monetario reale, non `null`, valore assente o codice di errore.

Il campionamento runtime:

1. trova la legge mediante la tupla;
2. estrae un intero uniforme sotto la somma delle masse;
3. seleziona il primo supporto con cumulativa strettamente maggiore.

Non interpola importi, non converte valuta e non annualizza numeri durante questa selezione. La probabilità esatta è il razionale `peso / somma`. [Campionamento](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_sampling.go:1).

#### Interi e razionali nella produzione

Il processo offline distingue la costruzione numerica dai valori finali runtime:

- curve di quantili e coefficienti sono trattati secondo l’aritmetica canonica prescritta;
- il profilo fast-demo usa 384 punti positivi derivati da quantili intermedi;
- la rettifica v3 prescrive Decimal a 96 cifre per specifici passaggi e moltiplicazione razionale esatta nei passaggi successivi indicati;
- la conversione al supporto monetario applica il ceiling all’importo valido, senza passare attraverso binary floating point;
- le masse finali sono interi canonici.

Il controllo del limite monetario precede la conversione: un valore sopra `4.294.967.295` non viene reso accettabile arrotondandolo.

Il JSON runtime conserva **gli interi finali**, non gli oggetti razionali della produzione. Una futura conversione di questo JSON deve preservare tali interi, senza ricostruirli dalle curve o da percentuali approssimate. [D2-B1 v2](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B1_v2.md:121), [rettifica aritmetica v3](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3.md:315).

---

## E. Catalogo fotografico

### E.1 Formato esterno consumato dal runtime

Percorso corrente:

```text
synthetic-world/photos/v1/catalog.json
```

| JSON Pointer | Tipo | Valori e controlli |
|---|---|---|
| `/format` | intero | Esattamente 1 |
| `/release` | stringa | Esattamente `v1` |
| `/selectionSeed` | stringa esadecimale | 64 caratteri, decodificati in 32 byte non tutti zero |
| `/faces` | array | Da 1 a 100.000 record |
| `/faces/k/id` | stringa | Unica; 1–128 caratteri ASCII alfanumerici, `_` o `-` |
| `/faces/k/gender` | stringa | `female` oppure `male` |
| `/faces/k/requestedAge` | intero | 21–75 |

Il catalogo è limitato a 32 MiB. Il decoder controlla il formato, rifiuta proprietà sconosciute e documenti multipli; la costruzione richiede almeno un volto utilizzabile per ciascun genere.

I record vengono ordinati per ID. Non è richiesta copertura di tutte le età. [Formato e validazione esterni](/Users/marco/src/krenalis/tools/fakedata/external_photo.go:99).

Per ogni record viene costruita una sola rendition runtime:

```text
synthetic-world/photos/v1/256/<id>.webp
```

con dimensioni dichiarate 256×256. Il caricamento del catalogo esterno non legge né verifica i byte dell’immagine.

#### Ruolo di `selectionSeed`

Il valore viene copiato nel checksum usato dal `FaceCatalog`. **Non è calcolato dal runtime come SHA del catalogo scaricato.**

Pertanto:

- entra nel determinismo della scelta;
- non dimostra l’integrità del JSON o dei WebP;
- la relazione tra quel seed e il contenuto pubblicato deve essere stabilita dal processo di distribuzione, non è verificata tramite hash del payload da questo decoder.

Il catalogo esterno non contiene hash, lunghezza in byte o metadati del modello generativo per ciascuna immagine. [Costruzione del catalogo esterno](/Users/marco/src/krenalis/tools/fakedata/external_photo.go:186).

### E.2 Selezione del PhotoID

Il selettore considera:

1. record generati e utilizzabili;
2. stesso genere della persona;
3. minima distanza assoluta tra `requestedAge` e **AnchorAge**;
4. scelta uniforme tra i record a pari distanza minima.

Non esiste una distanza massima oltre la quale il candidato viene scartato. I candidati seguono l’ordine normalizzato degli ID.

Lo stream `photo/select` include versione e checksum del catalogo oltre all’identità dello stream della persona. Non usa `CurrentAge` né `ReferenceDate`. [Selezione](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:287).

`PhotoID` identifica un elemento del catalogo, non una persona. La stessa fotografia può essere associata a più persone.

### E.3 Formato locale fakefacegen

Il formato locale comprende tre documenti distinti. [Tipi locali](/Users/marco/src/krenalis/tools/fakedata/fakefacegen.go:66).

#### `catalog.json`

| Proprietà | Tipo | Valore/ruolo corrente |
|---|---|---|
| `catalog_version` | intero | 1 |
| `seed` | `uint64` | Seed del produttore |
| `count` | intero | Numero di specifiche/asset |
| `spec_version` | stringa | `spec-v1` |
| `prompt_version` | stringa | `portrait-v1` |
| `model` | stringa | `gpt-image-2` |
| `quality` | stringa | `low` |
| `size` | stringa | `1024x1024` |
| `format` | stringa | `webp` |
| `background` | stringa | `opaque` |

#### `specs.json`

Array di `count` oggetti:

| Proprietà | Tipo | Ruolo |
|---|---|---|
| `id` | stringa | ID posizionale `face-%06d`, partendo da 1 |
| `age` | intero | Età richiesta, 21–75 |
| `sex` | stringa | `female`/`male`; convertito nella classificazione runtime |
| `skin_tone` | stringa | Specifica estetica |
| `face_shape` | stringa | Specifica estetica |
| `hair_color` | stringa | Specifica estetica |
| `hair_length` | stringa | Specifica estetica |
| `hair_texture` | stringa | Specifica estetica |
| `receding_hairline` | booleano | Specifica estetica |
| `facial_hair` | stringa opzionale | Specifica estetica |
| `glasses` | booleano | Specifica estetica |
| `natural_detail` | stringa opzionale | Specifica estetica |
| `clothing` | stringa | Specifica del ritratto |
| `environment` | stringa | Specifica del ritratto |
| `expression` | stringa | Specifica del ritratto |
| `lighting` | stringa | Specifica del ritratto |
| `photo_style` | stringa | Specifica del ritratto |

Il loader fakedata usa principalmente ID, età e sesso per il catalogo; non interpreta tutte le stringhe estetiche come tassonomie della persona sintetica. Non sarebbe corretto promuoverle automaticamente a fatti personali.

#### `manifest.json`

Oggetto con proprietà `assets`, array di record:

| Proprietà | Tipo | Ruolo/validazione |
|---|---|---|
| `id` | stringa | Identificatore unico |
| `file` | stringa opzionale | Per generated: `masters/<id>.webp` |
| `spec_id` | stringa | Deve riferire la specifica corrispondente |
| `spec_version` | stringa | Coerente con il catalogo |
| `prompt_version` | stringa | Coerente con il catalogo |
| `model` | stringa | Coerente con il catalogo |
| `quality` | stringa | Coerente con il catalogo |
| `size` | stringa | Coerente con il catalogo |
| `format` | stringa | Coerente con il catalogo |
| `synthetic` | booleano | Deve essere true |
| `status` | stringa | `generated`, `failed`, `missing` |
| `sha256` | stringa opzionale | Per generated: hash del master, 64 cifre hex minuscole |
| `bytes` | intero opzionale | Per generated: lunghezza positiva |
| `width` | intero opzionale | Per generated: 1024 |
| `height` | intero opzionale | Per generated: 1024 |
| `request_id` | stringa opzionale | Diagnostica del produttore |
| `error` | oggetto opzionale | `code` opzionale e `message` stringa |

Gli asset non generati restano rappresentabili come esiti del produttore ma non sono candidati fotografici utilizzabili. [Controlli di corrispondenza](/Users/marco/src/krenalis/tools/fakedata/fakefacegen.go:142).

### E.4 WebP, rendition e integrità locale

Il loader locale usa cinque rendition:

| Dimensione | Percorso locale |
|---:|---|
| 64 | `derived/64/<id>.webp` |
| 128 | `derived/128/<id>.webp` |
| 256 | `derived/256/<id>.webp` |
| 512 | `derived/512/<id>.webp` |
| 1024 | `masters/<id>.webp` |

Il codice verifica struttura/formato WebP, decodifica, dimensioni e hash; per il master confronta anche hash e lunghezza con il manifesto. Per le derivate calcola l’hash dai byte. [Mappatura e lettura immagini](/Users/marco/src/krenalis/tools/fakedata/fakefacegen.go:318).

Il checksum semantico del catalogo locale include:

- versione;
- hash dei byte di `catalog.json`;
- modalità completa o fixture selezionata;
- record ordinati con ID, stato, genere, età;
- hash delle cinque rendition per gli asset generati.

Il seed del produttore influisce indirettamente attraverso i byte di `catalog.json`; `request_id`, messaggi d’errore e descrizioni estetiche non sono direttamente componenti della selezione. [Identità locale](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:723).

Il riferimento locale pubblico ha forma:

```text
/photos/<checksum-semantico>/<dimensione>/<PhotoID>.webp
```

Per gli asset esterni l’integrazione seleziona la rendition 256 e usa `photoBaseURL.JoinPath(asset.Path)`. [Costruzione URL applicativa](/Users/marco/src/krenalis/core/simulated_system_generation.go:591).

### E.5 Asset locali e fixture: cosa è stato verificato

| Insieme locale | Seed | Record dichiarati generated | Età | Generi | File presenti |
|---|---:|---:|---|---|---|
| `tools/fakefacegen/output/catalog-test` | 726381 | 50 | 21–68 | 27 male, 23 female | 50 per ciascuna delle cinque rendition |
| `tools/fakefacegen/output/faces-v1` | 726381 | 1.250 | 21–75 | 598 male, 652 female | 1.250 per ciascuna rendition |

Sono stati verificati metadati e conteggi dei file, **non** nuovamente decodificati o certificati i byte delle immagini. Non è stata verificata quale raccolta sia attualmente pubblicata o configurata.

La fixture `fakefacegen-v1-selected` conserva metadati del catalogo da 50 record e soltanto le immagini selezionate `face-000013` e `face-000037`. Nel materiale locale queste hanno rispettivamente female/21 e male/68.

Le fixture dei test esterni usano anche ID uguali con età artificiali differenti: non sono evidenza delle proprietà delle foto reali. [Descrizione della fixture](/Users/marco/src/krenalis/tools/fakedata/testdata/fakefacegen-v1-selected/FIXTURE.md:1).

### E.6 Separazione delle responsabilità dei metadati

| Categoria | Informazioni |
|---|---|
| Necessarie alla generazione attuale | ID candidati, genere, età richiesta, utilizzabilità, ordine normalizzato, identità/versione di selezione |
| Distribuzione e validazione | Release, percorso/rendition, dimensioni, eventuali hash e lunghezze |
| Produttore fotografico | Prompt/spec version, modello, qualità, seed produttivo, descrizioni estetiche, request ID, errori |
| Fixture | Sottoinsiemi selezionati, età e seed artificiali dei test, dichiarazioni di copertura parziale |

Il formato esterno attuale contiene solo una parte di queste informazioni. Questo fatto non implica che ogni proprietà locale debba entrare nel futuro catalogo comune.

---

## F. Semantiche italiane incorporate nel Go

| Proprietà | Posizione attuale | Utilizzo | Natura | Rigidità da evitare e conservazione |
|---|---|---|---|---|
| Età 21–75 e 55 posizioni | `layer1_data.go`, modello L2 | Nascita, foto, fatti | Dataset più vincolo di protocollo | Non universali; conservare copertura e significato italiano |
| Ancoraggio 2026-01-01 | `layer1.go` | AnchorAge e fatti | Versione temporale del modello | Preservare il riferimento temporale |
| Cinque coorti nominali | `layer1_helpers.go` | Distribuzioni dei nomi | Classificazione nazionale | Non fissare cinque coorti per tutti; preservare confini inclusivi |
| 80 nomi per gruppo | Parser e contratto D1-B | Completezza dei gruppi | Cardinalità editoriale italiana | Conservare gli 800 record, senza generalizzare 80 |
| `female`/`male` | Nomi, persona, foto | Gruppi e candidati | Classificazione del modello attuale | Preservare i significati operativi; non dedurre una tassonomia universale |
| Genere equiprobabile | `layer1.go` | Generazione persona | Algoritmo | Non è stimato dai pesi demografici |
| ISTAT a sei cifre | CSV e parser | Identità comunale, CAP | Codifica nazionale | Preservare zeri e associazioni; non imporre ISTAT |
| Sigla di due lettere | CSV e parser | Indirizzo | Rappresentazione territoriale italiana | Non equipararla a qualsiasi Stato/provincia |
| Venti regioni, ordine Go | `socioeconomic_facts.go` | Assi L2 | Tassonomia territoriale | Preservare ogni ordinale e stringa |
| Collegamento regionale per testo | `facts()` | Da comune a L2 | Implementazione corrente | Conservare la relazione semantica, senza supporre gerarchie equivalenti |
| Miscela 95% popolazione/5% uniforme | `layer1.go` | Comune | Algoritmo | Distinta dai dati di popolazione |
| Bucket geografici da 128 | `layer1_data.go` | Campionamento | Algoritmo | Non sono unità geografiche |
| CAP di cinque cifre, liste operative fino a otto | Parser, dataset, test | Indirizzo | Codifica e produzione italiana | Conservare stringhe, ordine e relazione |
| Tipi stradali e suffissi | Costanti Go | Composizione indirizzo | Algoritmo nazionale | Non sono proprietà universali del lessico |
| Paese `IT` | Go | Address.Country | Dato nazionale incorporato | Conservare il significato italiano |
| 7 Education, 6 attività, 14 Industry, 10 famiglie, 4 livelli | Tipi, array e contratti | Vettori L2 | Tassonomie del modello | Non identificare gli ordinali con classificazioni straniere |
| Fasce dimensionali 0/1–9/10–49/50–249/250+ | Go e contratto | Company/Income | Classificazione derivata | Conservare soglie e asse `zero` |
| Euro interi annualizzati | Tipo `IncomeEUR`, Contract 5 | Reddito | Unità e significato economico | Nessuna conversione implicita di valuta |
| Età e regione come genitori dei fatti | `facts()` | Selezioni L2 | Grafo del modello | Conservare quali valori condizionano ciascuna distribuzione |
| Righe complete anche a monte di massa zero | Compilatore | Validazione del modello | Protocollo | Non eliminare celle solo perché attualmente non selezionate |
| Sentinel 65.535 | Compilatore Income | Fascia employee irraggiungibile | Codifica del protocollo | Preservare distinzione da zero monetario |
| Company universe e 40 indici Employer | `socioeconomic.go` | Employer | Algoritmo e stato derivato | Conservare le regole di associazione; non confondere con Groups |
| SHA dei CSV e fingerprint dei nodi | Layer 1 e identità L2 | Stream deterministici | Protocollo corrente | Documentarne gli effetti senza imporre ora la scelta futura |

Le regole possono avere natura mista. Per esempio, “cinque coorti” è una scelta del modello italiano resa rigida dal codice; il metodo di campionamento è un algoritmo. **Non segue che entrambe debbano essere trasformate in dati JSON.**

---

## G. Vincoli di conversione senza perdita

### G.1 Matrice dei contenuti da conservare

| Componente | Contenuto logico da preservare integralmente | Rappresentazione fisica potenzialmente diversa |
|---|---|---|
| Età | Tutte le 55 età, pesi interi, riferimento temporale e significato | Contenitore e nomi delle proprietà |
| Nomi | Genere, coorte, intervalli, display, atomi ASCII, pesi, appartenenza e ordine | Organizzazione dei gruppi |
| Cognomi | Testi Unicode, atomi distinti, pesi, ordine e identità | Struttura del documento |
| Località | Codici con zeri, denominazioni, sigle, regioni, popolazioni, relazioni | Modalità di rappresentare i collegamenti |
| CAP | Tutti i valori testuali, ordine interno, associazione al comune | Separatore `;` e struttura della lista |
| Strade | Frammenti originali, Unicode, pesi, ordine | Formato del lessico |
| Tassonomie L2 | Categorie, significati, ordinali e applicabilità | Collocazione delle definizioni |
| Partizioni d’età L2 | Otto partizioni distinte, limiti e mappa per età | Ridondanza fisica, purché ricostruibile esattamente |
| Pesi categorici | Interi originali, zeri, righe e assi | Appiattimento o organizzazione strutturale |
| EmployeeCount | Leggi per Industry, supporti e masse esatti | Contenitore delle leggi |
| Employer | Cap, tutti i fattori e relative combinazioni | Organizzazione matriciale |
| Income | Supporti EUR interi, masse, mappe, ID canonici, sentinel, condizionamenti | Struttura del documento, senza conversioni economiche |
| Identità del modello | Schema/versioni, contenuto canonico e collegamento alle evidenze | Metadati aggiuntivi o posizione fisica |
| Foto runtime | Identità dei candidati, genere, età, relazione alle immagini, discriminanti di selezione | Forma del catalogo |
| Foto di distribuzione | Rendition e integrità, quando presenti nell’asset sorgente | Organizzazione dei manifesti |
| Produzione offline | Provenienza e coefficienti esatti se inclusi nell’oggetto della conversione | Separazione dagli asset runtime |

Conservare solo le probabilità approssimate non è sufficiente: i pesi interi sono valori originali. Analogamente, conservare solo un cognome ASCII perde il display Unicode; conservare solo il nome di una categoria perde il suo collegamento ai vettori.

L’assenza di richiesta di retrocompatibilità degli output CSV permette di progettare diversamente il futuro protocollo. **Non autorizza a perdere i valori o il significato degli ordinamenti originali.**

### G.2 I sei SHA dei CSV

| Dataset | SHA-256 verificato |
|---|---|
| Età | `5abdc04bb817e5fc300ce71920c1dedf269551d3cc0065148264d26a39c84555` |
| Nomi | `ed249d8095d3133d90a662bb3dc4d592941d9aa1de1d476a8000b392761ed8db` |
| Cognomi | `d44f1859847b5c94066d20c0846fb16859827802bdf878d59e17dc26a046f96b` |
| Località | `27539c5025412c2f03b199cbd6fdea7b9bda8fb57f6d8665883c70eeaa7b05b6` |
| CAP | `0fed3e2822c13a54c4255d2b83fe0fd63aea27b432d63a682b28f64f4981426c` |
| Strade | `13a757c70266ad96f0c1f9d76ff32597995090009e860eec86a1dded5b5095ec` |

Sono convertiti da esadecimale ai **32 byte del digest** e inseriti come componenti degli stream, non come stringhe hex. [Componenti SHA](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:118).

| Componente | Stream diretto | Principali effetti indiretti |
|---|---|---|
| `age-data-sha256` | `identity/birth/age` | Nascita, coorte, età fotografica e fatti L2 |
| `first-name-data-sha256` | `identity/name/first` | Nome ed email |
| `last-name-data-sha256` | `identity/name/last` | Cognome ed email |
| `places-data-sha256` | Scelta uniforme/pesata del comune | Indirizzo, CAP applicabile, regione e fatti L2 |
| `postal-data-sha256` | `geography/place/postal` | CAP |
| `street-name-data-sha256` | `geography/address/fragment` | Frammento stradale |

La scelta del genere, il giorno nel periodo di nascita, il ramo geografico 95/5, tipo stradale, civico e suffisso hanno stream propri e non ricevono tutti gli SHA indiscriminatamente.

Il protocollo corrente è quindi sensibile ai byte dei CSV, ma con dipendenze separate per componente. I test di isolamento documentano questa separazione; non sono stati eseguiti durante lo studio. [Test delle dipendenze](/Users/marco/src/krenalis/tools/fakedata/layer1_data_test.go:15).

### G.3 Identità di dati, persona, World e Snapshot

| Identificatore | Componenti rilevanti |
|---|---|
| `PersonID` | Namespace e coordinata della persona |
| `CompanyID` | Namespace, tipo entità e coordinata Company |
| `WorldID` | Namespace, `italy`, `it-2026a`, `person-v1`, `world-v1`, WorldSeed |
| `SnapshotID` | WorldID, ReferenceDate, versione e checksum del catalogo fotografico |
| Root degli stream | WorldModelVersion e WorldSeed |
| Digest logico L2 | SHA-256 della ricostruzione canonica `C(L)` |
| `SocioeconomicWorldID` | WorldID di base, digest logico completo, ancoraggio, `socioeconomic-v1` |

Namespace e seed hanno responsabilità differenti: il namespace contribuisce agli ID, mentre il root degli stream usa versione del mondo e seed. I sei SHA CSV non vengono inseriti come sei campi autonomi in WorldID. [Costruzione World e Snapshot](/Users/marco/src/krenalis/tools/fakedata/layer1.go:168).

Gli stream sono ricreati per indirizzo deterministico: scope, tipo entità, coordinata, percorso e dipendenze. Non dipendono da un generatore globale avanzato nell’ordine delle richieste. [Stream](/Users/marco/src/krenalis/tools/fakedata/stream.go:25).

### G.4 Fingerprint logico socioeconomico

`canonicalModel()` ricostruisce il documento logico dai dati compilati:

- pesi mediante differenze delle cumulative;
- limiti d’età mediante transizioni dei bucket;
- proprietà in ordine canonico;
- interi esatti e LF finale.

Il digest completo identifica il modello. Separatamente vengono prodotti **15 fingerprint di nodo**, ciascuno sul sottoalbero posseduto dal nodo e, quando applicabile, sulla sua partizione d’età. Il fingerprint riguarda l’intera distribuzione del nodo, non solo la riga selezionata. [Canonicalizzazione e fingerprint](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_identity.go:58).

I 15 nodi sono:

```text
company/industry
company/employee-count
person/education
person/primary-activity
person/employee/job-family
person/employee/job-level
person/employee/employer
person/self-employed/job-family
person/self-employed/industry
person/income/employee
person/income/self-employed
person/income/unemployed
person/income/student
person/income/retired
person/income/inactive
```

I percorsi hanno il prefisso di protocollo `socioeconomic/v1/`. Ogni stream include il fingerprint del nodo e i genitori ordinali pertinenti, riportati nella matrice degli assi.

Non entrano come dipendenze aggiuntive:

- l’ID interno della legge Income selezionata;
- il nome del file di pubblicazione;
- l’hash del producer;
- l’intero manifesto delle evidenze.

Il digest globale identifica il World socioeconomico, ma **non viene usato al posto dei fingerprint locali come unico seed di tutte le selezioni**. [Identità del World L2](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_identity.go:227).

### G.5 Determinismo richiesto e scelte del protocollo corrente

**Necessario per il requisito futuro:** definire senza ambiguità input logici, configurazione, versione del motore, ordinamenti, interpretazione numerica, indirizzi degli stream e relazioni tra dati; evitare dipendenze dall’ordine delle richieste o da stato casuale condiviso.

**Scelte dell’attuale protocollo:** SHA dei byte CSV, nomi dei percorsi, codifica delle componenti, fingerprint per nodo, bucket geografici, specifici ordinali e versioni.

Questo studio non decide se il futuro Layer 1 debba usare fingerprint logici o discriminanti stabili indipendenti dai dati. Non propone modifiche ai checkpoint né ai contratti frozen. Il protocollo esistente resta descritto come riferimento fattuale. [Protocollo deterministico normativo](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md:1).

---

## H. Limiti di generalità internazionale

Le osservazioni seguenti sono **inferenze progettualmente rilevanti**, non definizioni di un nuovo schema.

| Caso | Assunzione italiana attuale | Informazione che un formato comune deve poter esprimere | Regola che può restare nazionale |
|---|---|---|---|
| Numero di coorti diverso | Cinque etichette riconosciute dal Go | Appartenenza dei nomi a un numero diverso di coorti | Modalità con cui il generatore sceglie la coorte |
| Intervalli differenti | Confini fissi 1950–2005 | Significato temporale preciso di ogni coorte | Interpretazione nazionale delle date |
| Nomi variabili per gruppo | 80 per genere/coorte | Cardinalità diversa senza perdere record | Politica di curazione |
| Età diverse | 21–75 e ancoraggio fisso | Copertura e riferimento temporale | Algoritmo di nascita |
| Codici geografici non ISTAT | Sei cifre | Identificatori testuali e relativo sistema | Validazioni del sistema nazionale |
| Gerarchie differenti | Comune, sigla, regione | Relazioni territoriali realmente presenti | Composizione nazionale dell’indirizzo |
| Regioni socioeconomiche diverse | Venti nomi Go | Asse territoriale con numero e categorie propri | Raccordo dal luogo della persona |
| Istruzione diversa | Sette categorie italiane del modello | Significati, ordine e corrispondenza con le distribuzioni | Raccordi nazionali dalle fonti |
| Industry e professioni diverse | 14 Industry, 10 famiglie, 4 livelli | Tassonomie e assi propri, senza equivalenze implicite | Classificazione nazionale |
| Dimensioni d’impresa diverse | Cinque fasce con soglie fisse | Soglie e significato delle fasce utilizzate | Derivazione dalla misura nazionale |
| Valute diverse | EUR interi | Valuta, unità, precisione e periodo del reddito | Modello economico e campionamento nazionale |
| Codici postali diversi | Cinque cifre e `;` | Valori testuali e relazione con le località | Validazione postale nazionale |
| Indirizzi diversi | Tipo+frammento+civico+suffisso | Contenuto del lessico e sue funzioni | Ordine e regole di composizione |
| Cataloghi fotografici diversi | Due generi, età 21–75 | Significato dei metadati e candidati disponibili | Politica di abbinamento |
| Matrici di forma diversa | Dimensioni codificate nel decoder | Assi, cardinalità e associazioni inequivocabili | Implementazione dei consumatori nazionali |

Provincia italiana, regione italiana e Stato di un’altra nazione non sono sostituibili soltanto perché occupano una posizione simile in un indirizzo.

Anche categorie con nomi apparentemente comuni — per esempio “tertiary” o “manager” — richiedono conservazione del significato contrattuale. La possibilità di riutilizzarle altrove è una valutazione successiva, non un fatto dimostrato dall’implementazione italiana.

---

## I. Lacune e informazioni non verificabili

### I.1 Cosa è completo

Sono disponibili per lo studio:

- tutti i valori dei sei CSV;
- struttura completa del decoder e del compilatore socioeconomico;
- tutti i valori del modello logico canonico locale;
- classificazioni, ordinali, formule e dipendenze runtime;
- contratti frozen pertinenti;
- metadati di pubblicazione e crosswalk D2-B2;
- formati fotografici esterno e locale;
- metadati di due raccolte fotografiche locali.

**Non esiste quindi una lacuna dei valori numerici canonici che obblighi a limitarsi alla sola struttura del decoder.**

### I.2 Cosa impedisce una prova di equivalenza della futura conversione

La conversione non esiste ancora e non è stata eseguita. Per convalidarla serviranno:

| Materiale | Necessità |
|---|---|
| Futuro asset italiano nella nuova rappresentazione | Confrontare tutti i valori e le relazioni |
| Regole precise di interpretazione della proposta | Verificare assi, unità, classificazioni e ordinamenti |
| Sei CSV originali e modello canonico attuale | Baseline completa del contenuto |
| Contratti e tassonomie Go attuali | Baseline dei significati non presenti nei file |
| Specifica del futuro protocollo deterministico | Verificare il nuovo determinismo, senza imporre gli output CSV precedenti |

Questo dossier non dichiara dimostrata un’equivalenza che richiede ancora il secondo termine del confronto.

### I.3 Lacune che non bloccano la descrizione strutturale

| Lacuna | Conseguenza |
|---|---|
| Asset runtime remoto non scaricato | Non è dimostrato che l’endpoint pubblicato contenga gli stessi byte del canonico locale |
| Producer dichiarato non canonico contro report canonico | Serve chiarimento per una riproduzione auditabile della produzione |
| Input originali pinned della produzione non nuovamente verificati | Non è dimostrata la riproducibilità completa dalle fonti |
| Materiali offline D1 non interamente ricostruiti | Provenienza ricostruibile solo nei limiti dei documenti presenti |
| Review postali con stati discordanti | Catena di approvazione documentale incompleta |
| Catalogo fotografico esterno effettivamente pubblicato non acquisito | Non è stabilito il legame tra release esterna, selectionSeed e raccolte locali |
| Byte delle immagini locali non rivalidati in questo studio | Conteggi e metadati non equivalgono a una nuova certificazione d’integrità |

Per convalidare anche la parte fotografica pubblicata servirebbero il catalogo esterno effettivo e, se l’obiettivo include integrità/equivalenza delle immagini, i relativi WebP e un manifesto d’identità verificabile.

Gli input originali e il comando canonico di produzione servono a dimostrare la **riproducibilità della produzione**, ma non sono indispensabili per progettare una rappresentazione senza perdita dei valori logici già disponibili.

Nessuna classificazione statunitense è stata ricercata o inventata.

---

## J. Riferimenti tecnici essenziali

| Riferimento | Ruolo |
|---|---|
| [Guida progetto](/Users/marco/src/krenalis/SYNTHETIC_WORLD_GUIDA_PROGETTO.md:1) | Orientamento e stato storico del progetto |
| [Ricognizione External Assets](/Users/marco/src/krenalis/SYNTHETIC_WORLD_EXTERNAL_ASSETS_RICOGNIZIONE.md:1) | Punti d’ingresso e distinzione degli asset |
| [Fonti CSV](/Users/marco/src/krenalis/tools/fakedata/data/it-2026a/SOURCES.md:1) | Provenienza dei dati italiani |
| [Caricatore Layer 1](/Users/marco/src/krenalis/tools/fakedata/layer1_data.go:21) | Hash, tipi, parser, cardinalità e indici |
| [Generatore Layer 1](/Users/marco/src/krenalis/tools/fakedata/layer1.go:13) | Date, nomi, geografia, contatti e indirizzi |
| [Helper Layer 1](/Users/marco/src/krenalis/tools/fakedata/layer1_helpers.go:26) | Calcolo età e confini effettivi delle coorti |
| [Contratto D1-B](/Users/marco/src/krenalis/L1-D1-B_Italian_Curated_Plausibility_Data_Contract.md:1) e [rettifica](/Users/marco/src/krenalis/L1-D1-B_Contract_Rectification.md:1) | Curazione, Unicode, identità, ordine e pesi |
| [Modello canonico](/Users/marco/src/krenalis/D2B2-logical-assets-v0.json:1) | Valori numerici reali di riferimento |
| [Decoder L2](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_json.go:16) | Albero chiuso, tipi e limiti del JSON |
| [Compilatore L2](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_model_compile.go:12) | Canonicalità numerica, dimensioni e riferimenti |
| [Tipi socioeconomici](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_types.go:7) | Valori pubblici e strutture dei fatti |
| [Fatti socioeconomici](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_facts.go:12) | Ordinali, collegamento geografico e formule degli assi |
| [World socioeconomico](/Users/marco/src/krenalis/tools/fakedata/socioeconomic.go:12) | Company universe e indici Employer |
| [Campionamento L2](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_sampling.go:1) | Stream dei nodi e selezione discreta |
| [Identità L2](/Users/marco/src/krenalis/tools/fakedata/socioeconomic_identity.go:58) | Ricostruzione canonica, fingerprint e World ID |
| [Contract 2](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:1) | Company ed Employer |
| [Contract 3](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:1) | Tassonomie, istruzione e occupazione |
| [Contract 4](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:1) | Attività e dipendenze del modello |
| [Contract 5](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:1) | Significato economico e rappresentazione Income |
| [D2-A frozen](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md:1) | Modello numerico, assi, leggi e canonicalità |
| [Protocollo deterministico](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md:1) | Identità, stream e semantiche frozen |
| [Amendment caricamento esterno](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md:1) | Acquisizione del modello logico tramite External Assets |
| [Publication D2-B2](/Users/marco/src/krenalis/D2B2-publication-v3.json:1), [production report](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PRODUCTION_REPORT.md:1), [review canonica](/Users/marco/src/krenalis/SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_ASTRA_CANONICAL_ASSET_REVIEW.md:1) | Identità e attestazioni della produzione locale |
| [Catalogo fotografico esterno](/Users/marco/src/krenalis/tools/fakedata/external_photo.go:99) | Formato minimale e normalizzazione |
| [Loader fakefacegen](/Users/marco/src/krenalis/tools/fakedata/fakefacegen.go:66) | Formati locali, manifesti e WebP |
| [Selezione fotografica](/Users/marco/src/krenalis/tools/fakedata/layer1_photo.go:287) | Abbinamento, checksum e rendition |

**Conclusione:** i dati italiani necessari alla futura mappatura sono localmente disponibili, compreso il modello socioeconomico canonico. La conservazione senza perdita richiede di trattare come parte del contenuto anche tassonomie, ordinali, unità, confini temporali e relazioni oggi espresse nel Go o nei contratti. Le lacune residue riguardano soprattutto pubblicazione effettiva e riproducibilità delle produzioni offline; non giustificano l’uso di fixture al posto dei valori canonici.
