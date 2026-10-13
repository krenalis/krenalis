# Specifica candidata consolidata — Synthetic World JSON internazionali v1

**Status: CANDIDATE — Not approved / Not frozen.**

La direzione architetturale precedente resta confermata. Questo consolidamento modifica quattro punti: **geografia leggera**, **raccordo territoriale indipendente dai file fisici**, **identità logiche definite a livello di byte** e **separazione tra identità della selezione fotografica e contenuto del catalogo**.

Le prescrizioni seguenti sono **nuove decisioni candidate**, salvo i richiami espliciti ai contratti frozen. Non costituiscono un nuovo freeze.

La successiva decisione del proprietario ritira la generazione legacy: la nuova baseline italiana usa soltanto `generation-v5` per la generazione non vuota, con dati JSON acquisiti tramite External Assets; `empty-v1` resta supportato. Non sono richiesti retrocompatibilità degli output o mantenimento operativo di v3/v4 e dei CSV embedded. I contratti frozen restano intatti come autorità delle semantiche storiche e delle regole condivise richiamate; questa candidatura disciplina un percorso successivo. La review precedente resta evidenza della candidatura allora esaminata, senza attestare questa nuova decisione.

Ho verificato in sola lettura gli allegati originali: numerosità e SHA-256 dei sei CSV coincidono con le attestazioni; il modello contiene i 12.194 pesi categorici, le quattordici leggi EmployeeCount e i 76.160 riferimenti Income previsti. Ho inoltre verificato che, **per il JSON allegato**, i byte originali coincidono con la serializzazione canonica descritta dal protocollo. Questo non equivale ad avere eseguito `canonicalModel()` o il compilatore Go.

---

# A. Decisioni architetturali definitive proposte

| Decisione | Specifica candidata | Motivazione |
|---|---|---|
| Famiglia dei formati | Quattro schemi chiusi: base, socioeconomico, catalogo fotografico e descrittore | Separazione dei domini senza un contenitore generico |
| Geografia | Attributi amministrativi nella località; codice del territorio socioeconomico separato | Elimina tabelle di aree e gerarchie non presenti nei dati originali |
| Territori socioeconomici | Tassonomia autonoma nel modello; raccordo mediante `(semanticId, code)` | Il modello può essere validato senza acquisire la geografia |
| Distribuzioni | Assi prescritti, categorie ordinate, vettori densi e leggi discrete | Conserva gli array italiani e rende esplicita la loro interpretazione |
| Partizioni d’età | Otto oggetti indipendenti; soltanto `lowerBounds` | `compiled` si ricostruisce univocamente |
| Identità Layer 1 | Sei fingerprint logici separati | Conserva l’isolamento delle famiglie senza dipendere dai byte CSV |
| Identità Layer 2 | Proiezione nel modello originale e riutilizzo dei fingerprint frozen | Nessuna modifica del modello numerico o del protocollo stocastico |
| Fotografie | Fingerprint della selezione distinto dal digest completo del catalogo | Cambiare un percorso WebP non deve cambiare il volto selezionato |
| Acquisizione | Descrittore piccolo sopra il resolver External Assets esistente | Coordina asset indipendenti senza un secondo sistema di distribuzione |
| Runtime | Nuovo ingresso versionato per dati già acquisiti; adattamento italiano | Nessun interprete statistico, plugin nazionale universale o generazione remota |

**Criterio dati/regole.** Un valore appartiene al JSON quando descrive il contenuto del dataset, una classificazione, un supporto, un peso o una relazione necessaria a interpretare quei dati. Appartiene al Go quando prescrive un algoritmo di generazione.

Di conseguenza, rimangono nel Go nascita, miscela geografica 95/5, bucket di campionamento, pattern dei contatti, composizione degli indirizzi, formula Employer, applicabilità e ordine di valutazione dei nodi. Cap Employer, fattori, soglie delle fasce, coorti e distribuzioni restano dati dichiarati e verificati: non sono formule eseguibili.

---

# B. Contratti JSON internazionali completi

## B.1 Convenzioni comuni

### Obbligatorietà e oggetti chiusi

Nelle tabelle:

- **R** significa obbligatorio.
- **O** significa facoltativo; l’assenza conserva il proprio significato e non viene sostituita con un valore predefinito.
- **`null` è vietato ovunque**, salvo negli elementi specificamente ammessi di `income.employee.tupleMap`.
- Ogni oggetto ammette esclusivamente le proprietà elencate.
- L’unica mappa a chiavi libere del documento base è la mappa tipizzata dei codici postali.

Non sono ammessi documenti parziali nel caricamento operativo. Gli estratti della sezione E sono esclusivamente illustrativi.

### Tipi comuni

| Tipo della specifica | Tipo JSON e vincoli |
|---|---|
| `Token` | Stringa ASCII di 1–128 byte, conforme a `[A-Za-z0-9][A-Za-z0-9._:-]*` |
| `Text` | Stringa Unicode non vuota, composta da valori scalari validi; senza controlli U+0000–U+001F e U+007F |
| `Code` | `Text`, confrontato esattamente e senza alias; può contenere zeri iniziali, spazi interni, apostrofi e caratteri non ASCII |
| `Country` | Stringa conforme a `[A-Z]{2}`; il profilo implementato ammette `IT` |
| `Date` | Stringa `YYYY-MM-DD`, data gregoriana valida, anni 0001–9999 |
| `U32` | Intero JSON nell’intervallo 0–4.294.967.295 |
| `U64` | Intero JSON nell’intervallo 0–18.446.744.073.709.551.615 |
| `PositiveU64` | `U64` maggiore di zero |
| `SHA256` | Stringa di esattamente 64 cifre esadecimali minuscole |
| `AssetPath` | Percorso relativo ASCII, definito sotto |
| `ASCIIAtom` | Stringa conforme a `[a-z0-9]+(?:-[a-z0-9]+)*` |

Le eventuali restrizioni nazionali sui testi si aggiungono a queste regole. Il decoder non elimina spazi, non cambia maiuscole, non normalizza Unicode e non corregge codici.

Tutti i numeri devono avere la forma lessicale:

```text
0
oppure
[1-9][0-9]*
```

Sono vietati `-0`, segni, zeri iniziali, frazioni, esponenti e numeri racchiusi tra virgolette quando è richiesto un intero.

### Percorsi degli asset

`AssetPath` è relativo alla radice di ciascuna origin di `KRENALIS_EXTERNAL_ASSETS_URLS`, **non alla directory del descrittore**.

È composto da segmenti non vuoti separati da `/`. Ogni segmento usa soltanto lettere ASCII, cifre, `.`, `_`, `-`; i segmenti `.` e `..` sono vietati. Sono vietati slash iniziale/finale, segmenti vuoti, percent-encoding, query, fragment, backslash e URL assoluti.

Questa è una convenzione di indirizzamento del formato, non un nuovo resolver.

### Tassonomia comune

La struttura `Taxonomy` è:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `semanticId` | `Token` | R | Identità versionata della classificazione |
| `categories` | Array di `Category` | R | Almeno una categoria; codici unici; ordine significativo |

`Category` contiene:

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `code` | `Code` | R | Identificatore della categoria nella tassonomia |
| `definition` | `Text` | O | Descrizione esplicativa; non viene interpretata per dedurre equivalenze |

**Correzione rispetto alla proposta precedente:** la descrizione testuale non è obbligatoria. Il significato autorevole è quello del binding versionato identificato da `semanticId`; la descrizione non può ridefinirlo.

L’ordinale numerico è la posizione nell’array, a partire da zero. Non viene serializzato un secondo campo `ordinal`.

Il ruolo verificabile di `semanticId` è precisato nella sezione C.

---

## B.2 Documento dei dati di base

Schema:

```text
sw-base-data-v1
```

### Oggetto principale

| Campo | Tipo | Presenza | Significato e vincoli |
|---|---|---:|---|
| `schema` | Stringa letterale | R | Esattamente `sw-base-data-v1` |
| `release` | `Token` | R | Pubblicazione immutabile di questo documento |
| `profile` | `Token` | R | Profilo nazionale di dati e compatibilità |
| `country` | `Country` | R | Contesto nazionale della generazione, non cittadinanza |
| `anchorDate` | `Date` | R | Riferimento delle età |
| `genders` | `Taxonomy` | R | Classificazione usata dai gruppi nominali |
| `ages` | Array di `AgeWeight` | R | Non vuoto; età strettamente crescenti |
| `cohorts` | Array di `Cohort` | R | Non vuoto; intervalli ordinati e non sovrapposti |
| `firstNames` | Array di `FirstNameGroup` | R | Copertura completa del prodotto generi × coorti |
| `lastNames` | Array di `NameEntry` | R | Non vuoto; ordine significativo |
| `geography` | `Geography` | R | Località, attributi amministrativi, raccordo territoriale e CAP |
| `streetNames` | `StreetLexicon` | R | Lessico nazionale ordinato |

Il profilo italiano candidato è:

```text
it-2026a-base-v1
```

### Età

`AgeWeight`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `age` | `U32` | R | Età compiuta all’ancoraggio |
| `weight` | `PositiveU64` | R | Massa relativa esatta |

Il totale dei pesi deve essere rappresentabile in `U64`. Non si applica riduzione per MCD.

Lo schema ammette un insieme di età non necessariamente consecutivo. Il profilo italiano richiede precisamente le 55 età 21–75 e i relativi dati originali della release.

La compatibilità tra età, data di ancoraggio e algoritmo nazionale di nascita appartiene alla validazione del profilo, non viene dedotta soltanto dal tipo `U32`.

### Coorti

`Cohort`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `id` | `Token` | R | Unico nel documento |
| `fromYear` | Intero 1–9999 | R | Primo anno incluso |
| `toYear` | Intero 1–9999 | R | Ultimo anno incluso; non inferiore a `fromYear` |

L’array è ordinato per `fromYear` crescente. Per coorti consecutive deve valere:

\[
\text{precedente.toYear}<\text{successiva.fromYear}.
\]

Lo schema non interpreta sintatticamente `id` e non impone contiguità universale. Il profilo deve dimostrare che **ogni anno di nascita generabile appartiene a una e una sola coorte**. Una lacuna raggiungibile causa errore prima della generazione.

Per l’Italia restano i cinque intervalli inclusivi 1950–1964, 1965–1979, 1980–1989, 1990–1999 e 2000–2005. La nascita effettivamente generabile va dal 2 gennaio 1950 al 1° gennaio 2005: la copertura si verifica su questo dominio, non sottraendo semplicemente le età all’anno 2026. SYNTHETIC_WORLD_MODELLO_ITALIAN…

### Nomi propri e cognomi

`FirstNameGroup`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `gender` | `Code` | R | Codice presente in `genders.categories` |
| `cohort` | `Token` | R | ID presente in `cohorts` |
| `entries` | Array di `NameEntry` | R | Non vuoto; ordine significativo |

Ogni coppia `(gender, cohort)` compare esattamente una volta. L’ordine dei gruppi è il prodotto cartesiano: generi nell’ordine dichiarato, poi coorti nell’ordine dichiarato.

`NameEntry`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `display` | `Text` | R | Testo originale visualizzato |
| `asciiAtom` | `ASCIIAtom` | O | Obbligatorio nel profilo italiano |
| `weight` | `PositiveU64` | R | Massa relativa esatta |

Regole comuni:

- `display` è unico all’interno di ciascun gruppo nominale e nell’intero elenco dei cognomi.
- La stessa voce può comparire in coorti o generi diversi.
- Atomi uguali associati a display differenti sono ammessi.
- La somma dei pesi di ciascuna distribuzione deve essere positiva e rappresentabile in `U64`.
- L’ordine delle voci è quello del documento; il decoder non le riordina.
- Un atomo assente non viene generato automaticamente. Il profilo deve prevedere come il generatore gestisce tale assenza oppure rifiutarla.

Per l’Italia si applicano anche unicità e ordine secondo la chiave:

\[
D(s)=NFC(DefaultFullCaseFold_{\text{Unicode16}}(NFC(s))).
\]

Questa funzione serve alla verifica delle chiavi, non alla trasformazione dei dati durante il caricamento. Le collisioni degli atomi rimangono ammesse e non generano suffissi. La rettifica distingue espressamente queste due identità. L1-D1-B_Contract_Rectification(… L1-D1-B_Contract_Rectification

### Geografia

`Geography`:

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `placeCodeSystem` | `Token` | R | Identità versionata del sistema degli identificatori locali |
| `territorySystem` | `Token` | R | Identità della classificazione territoriale usata nel raccordo socioeconomico |
| `postalCodeSystem` | `Token` | R | Sistema/raccolta postale rappresentata |
| `places` | Array di `Place` | R | Non vuoto; ID unici; ordine significativo |
| `postalCodes` | Mappa `Code → array di Code` | R | Esattamente una chiave per ciascuna località |

`Place`:

| Campo | Tipo | Presenza | Significato e vincoli |
|---|---|---:|---|
| `id` | `Code` | R | Identificatore in `placeCodeSystem` |
| `name` | `Text` | R | Denominazione della località |
| `population` | `U64` | R | Conteggio di popolazione; zero rappresentabile nello schema comune |
| `administrative` | Array di `AdministrativeAttribute` | R | Può essere vuoto; ordine conservato |
| `territory` | `Code` | R | Codice in `territorySystem` |

`AdministrativeAttribute`:

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `system` | `Token` | R | Identifica congiuntamente il tipo amministrativo e il sistema di identificazione |
| `code` | `Code` | O | Codice amministrativo |
| `name` | `Text` | O | Denominazione amministrativa |

Almeno uno tra `code` e `name` deve essere presente. In una località non possono comparire due attributi con lo stesso `system`.

L’array è una sequenza di attributi, **non una dichiarazione implicita di parentela**. Il profilo nazionale stabilisce quali attributi servono all’indirizzo. Il formato non deduce una gerarchia dal loro ordine.

Quando nel documento viene ripetuta la stessa coppia `(system, code)` con un `name` esplicito, le denominazioni esplicite devono coincidere. Questo evita riferimenti contraddittori senza creare una tabella autonoma di aree.

Per l’Italia: popolazioni positive, 7.896 località, ID ISTAT a sei cifre in ordine strettamente crescente e un attributo provinciale/UTS per località.

### Codici postali

`postalCodes` è una mappa tipizzata. Le chiavi devono coincidere esattamente con l’insieme degli ID di `places`: nessuna località sconosciuta e nessuna chiave mancante.

Ogni valore è un array di codici postali:

- stringhe esatte, inclusi zeri iniziali e spazi eventualmente previsti dalla codifica;
- senza duplicati nella stessa località;
- con ordine significativo, conservato dal decoder;
- non necessariamente unici tra località diverse.

Lo schema comune consente `[]` soltanto come rappresentazione esplicita di “nessun codice postale utilizzabile”. Il profilo deve ammettere questo stato e avere una regola di generazione corrispondente. Non equivale a una chiave mancante.

Il profilo italiano richiede liste non vuote, CAP di cinque cifre, ordine strettamente crescente e le associazioni originali; la release attuale contiene al massimo otto CAP per comune. Non è dichiarato un inventario postale completo o autoritativo. SYNTHETIC_WORLD_MODELLO_ITALIAN…

### Lessico stradale

`StreetLexicon`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `kind` | Enumerazione | R | `fragment` oppure `full-name` |
| `entries` | Array di `StreetEntry` | R | Non vuoto; ordine significativo |

`StreetEntry` contiene soltanto `text: Text` e `weight: PositiveU64`, entrambi obbligatori.

Le voci devono avere testi unici; il totale dei pesi deve essere rappresentabile in `U64`. Il profilo italiano richiede `fragment`, 400 voci e l’ordine/identità D1-B.

Non sono presenti matrice di compatibilità con i tipi stradali, civici, suffissi o regole di composizione.

---

## B.3 Documento socioeconomico

Schema:

```text
sw-socioeconomic-data-v1
```

### Oggetto principale

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `schema` | Stringa letterale | R | `sw-socioeconomic-data-v1` |
| `release` | `Token` | R | Pubblicazione immutabile del documento |
| `profile` | `Token` | R | Profilo numerico e semantico nazionale |
| `country` | `Country` | R | Contesto nazionale |
| `anchorDate` | `Date` | R | Riferimento dei fatti socioeconomici |
| `sourceModel` | `SourceModel` | O | Provenienza della rappresentazione convertita |
| `ageDomain` | `AgeDomain` | R | Dominio interpretato dalle partizioni |
| `dimensions` | `Dimensions` | R | Sette classificazioni ordinate |
| `agePartitions` | `AgePartitions` | R | Otto partizioni indipendenti |
| `companyCount` | `PositiveU64` | R | Cardinalità dell’universo Company |
| `incomeUnit` | `IncomeUnit` | R | Significato e unità dei supporti Income |
| `models` | `Models` | R | Distribuzioni e fattori tipizzati |

Profilo italiano:

```text
it-2026a-socioeconomic-v1
```

`SourceModel` contiene esclusivamente:

| Campo | Tipo | Presenza |
|---|---|---:|
| `schema` | `Token` | R |
| `byteSha256` | `SHA256` | R |

Per questa conversione italiana è obbligatorio e identifica i **byte dell’originale**, non un digest Go presunto:

```text
schema = SW-L2-D2B2-logical-v0
byteSha256 = f049103241e6e43ffb5dd4a1247b64b2d0ca1bff787950f25284b5b435fd21a8
```

Non contiene URL e non provoca acquisizioni aggiuntive. Non è una prova sostitutiva della proiezione.

`AgeDomain` contiene `minimum: U32` e `maximum: U32`, obbligatori, con `minimum ≤ maximum`.

### Dimensioni

`Dimensions` contiene esattamente:

| Campo | Struttura |
|---|---|
| `education` | `Taxonomy` |
| `activity` | Tassonomia di attività |
| `industry` | `Taxonomy` |
| `jobFamily` | `Taxonomy` |
| `jobLevel` | `Taxonomy` |
| `territory` | `Taxonomy` |
| `sizeBand` | Tassonomia dimensionale Company |

Tutti sono obbligatori.

La tassonomia di attività usa `semanticId` e `categories`, ma ciascuna categoria aggiunge:

```text
branch: employee | selfEmployed | unemployed | student | retired | inactive
```

`branch` è obbligatorio. Ogni categoria ha un solo ramo; tutti i sei rami devono essere rappresentati.

Si ammettono più categorie nazionali associate allo stesso ramo. Questo non aggiunge un condizionamento: due categorie associate allo stesso ramo non autorizzano leggi Income differenti in base alla sottocategoria, salvo gli altri genitori già previsti.

La tassonomia `sizeBand` aggiunge a ciascuna categoria:

```text
minimumEmployeeCount: U32
```

Sono richieste almeno due categorie. I limiti sono strettamente crescenti; i primi due sono 0 e 1. La prima fascia rappresenta quindi **soltanto zero**. Le successive coprono i conteggi positivi fino a `MaxUint32`.

`positiveSizeBand` non è una dimensione serializzata: è la sequenza di `sizeBand.categories` privata della prima categoria.

### Partizioni d’età

`AgePartitions` contiene esattamente otto campi obbligatori:

```text
education
activity
employeeLevel
selfEmployedIncome
unemployedIncome
studentIncome
retiredIncome
inactiveIncome
```

Ogni campo è un oggetto chiuso contenente soltanto:

```text
lowerBounds: array non vuoto di U32
```

Non sono ammessi `compiled`, una partizione employee Income, alias o riferimenti ad altre partizioni.

### Unità Income

`IncomeUnit`:

| Campo | Tipo | Presenza | Valori |
|---|---|---:|---|
| `currency` | Stringa `[A-Z]{3}` | R | Codice della valuta del modello |
| `decimalPlaces` | Intero | R | 0–9 |
| `period` | Stringa letterale | R | `year` |
| `basis` | Stringa letterale | R | `gross-personal-monetary-annualized` |

Un supporto `v` rappresenta esattamente:

\[
v \times 10^{-\text{decimalPlaces}}
\]

unità della valuta dichiarata per anno annualizzato alle condizioni dell’ancoraggio.

La famiglia conserva il concetto di influsso personale monetario non negativo, prima delle imposte e contribuzioni personali; per le attività economiche si tratta della parte attribuibile alla persona dopo i costi operativi. Non rappresenta fatturato, patrimonio, risorse familiari o risultato contabile netto firmato. SYNTHETIC_WORLD_V1_LAYER2_CONTR…

Per l’Italia sono obbligatori EUR e precisione zero. I supporti già finali non attraversano nuovamente annualizzazione, ceiling o conversione monetaria.

### Modelli e strutture annidate

`Models` contiene esattamente questi dieci campi obbligatori:

| Campo | Tipo |
|---|---|
| `companyIndustry` | `CategoricalDistribution` |
| `employeeCount` | `CountLaws` |
| `education` | `CategoricalDistribution` |
| `primaryActivity` | `CategoricalDistribution` |
| `employeeJobFamily` | `CategoricalDistribution` |
| `employeeJobLevel` | `CategoricalDistribution` |
| `selfEmployedJobFamily` | `CategoricalDistribution` |
| `selfEmployedIndustry` | `CategoricalDistribution` |
| `employer` | `EmployerModel` |
| `income` | `IncomeModels` |

Le strutture sono:

| Oggetto | Campi obbligatori, e nessun altro |
|---|---|
| `CategoricalDistribution` | `axes: array di stringhe`, `outcome: stringa`, `weights: array di U32` |
| `CountLaws` | `axes: array di stringhe`, `laws: array di DiscreteLaw` |
| `DiscreteLaw` | `values: array di U32`, `weights: array di U32` |
| `FactorMatrix` | `axes: array di stringhe`, `values: array di interi 1–4095` |
| `EmployerModel` | `workforceCap: intero 1–65535`, `industryFactors: FactorMatrix`, `sizeFactors: FactorMatrix` |
| `IncomeBranch` | `axes: array di stringhe`, `tupleMap: array di riferimenti`, `laws: array non vuoto di DiscreteLaw` |

`IncomeModels` contiene esattamente sei `IncomeBranch` obbligatori:

```text
employee
selfEmployed
unemployed
student
retired
inactive
```

Un riferimento di `tupleMap` è un `U32` minore della lunghezza di `laws`, oppure il solo `null` ammesso dalla sezione D.

Le sequenze di `axes` e i valori di `outcome` **non sono liberi**: sono fissati dalla tabella D.1.

Le cardinalità internazionali derivano dalle dimensioni dichiarate. I massimi operativi italiani del decoder e di D2-A restano vincoli del profilo/consumatore italiano, non vengono trasferiti indiscriminatamente nello schema internazionale.

---

## B.4 Catalogo fotografico

Schema:

```text
sw-photo-catalog-v1
```

Non aggiungo un `country` o un profilo nazionale al catalogo: deve poter essere condiviso. In questa versione dello schema il metodo di selezione è unico.

### Oggetto principale

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `schema` | Stringa letterale | R | `sw-photo-catalog-v1` |
| `release` | `Token` | R | Pubblicazione immutabile del catalogo completo |
| `genderTaxonomy` | `Taxonomy` | R | Classificazione dei candidati |
| `selection` | `PhotoSelection` | R | Discriminanti espliciti della selezione |
| `faces` | Array di `Face` | R | Non vuoto; ID unici |

`PhotoSelection`:

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `version` | `Token` | R | Discriminante versionato della serie di selezione |
| `seed` | `SHA256` | R | 32 byte di seed, non tutti zero |

`selection.version` **non seleziona un algoritmo arbitrario**. La sua funzione è quella del discriminante di versione già utilizzato dalla selezione. Il metodo è fissato dal nuovo contratto. Nel runtime precedente il discriminante di selezione è `fakefacegen-v1`, distinto dalla release `v1`. La migrazione descritta in H.3 copia la release in `selection.version` come scelta esplicita della nuova baseline; questa copia non conserva il componente di versione dello stream precedente.

`Face`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `id` | Stringa | R | 1–128 caratteri ASCII alfanumerici, `_` o `-`; unica |
| `gender` | `Code` | R | Presente nella tassonomia del catalogo |
| `requestedAge` | `U32` | R | Età richiesta al produttore, non età inferita dall’immagine |
| `renditions` | Array di `Rendition` | R | Non vuoto |

`Rendition`:

| Campo | Tipo | Presenza | Vincoli |
|---|---|---:|---|
| `width` | `U32` positivo | R | Larghezza dichiarata |
| `height` | `U32` positivo | R | Altezza dichiarata |
| `path` | `AssetPath` | R | Deve terminare con `.webp` |
| `integrity` | `ImageIntegrity` | O | Dichiarazione di integrità dell’immagine |

`ImageIntegrity` contiene `sha256: SHA256` e `bytes: PositiveU64`, entrambi obbligatori quando l’oggetto è presente.

Per ogni volto la coppia `(width, height)` è unica. Se lo stesso percorso è riferito più volte nel catalogo, dimensioni e metadati d’integrità eventualmente dichiarati devono essere coerenti.

### Candidati e ordinamento

Ogni elemento di `faces` è un candidato utilizzabile dichiarato dal produttore della release. Non ci sono stati `failed`, `missing`, flag di esclusione o record diagnostici.

Un candidato malformato causa il rifiuto del catalogo; non viene saltato silenziosamente.

Il loader normalizza soltanto:

- `faces`, per `id` ASCII crescente;
- `renditions`, per `(width, height)` numericamente crescente.

Prima dell’ordinamento vengono rifiutati i duplicati.

La disponibilità HTTP del WebP non determina dinamicamente l’insieme dei candidati. Altrimenti la rete influenzerebbe la selezione.

Il catalogo metadata-only non dimostra da solo che le immagini esistano o che le dichiarazioni di integrità siano vere. La verifica dei byte, quando richiesta, è un’attività di pubblicazione/acquisizione delle immagini distinta dal caricamento dei metadati.

---

## B.5 Descrittore di release nazionale

Schema:

```text
sw-country-release-v1
```

| Campo | Tipo | Presenza | Significato |
|---|---|---:|---|
| `schema` | Stringa letterale | R | `sw-country-release-v1` |
| `release` | `Token` | R | Identità della release nazionale coordinata |
| `country` | `Country` | R | Paese del pacchetto |
| `base` | `AssetReference` | R | Documento base atteso |
| `socioeconomic` | `AssetReference` | R | Documento socioeconomico atteso |
| `photos` | `AssetReference` | R | Catalogo fotografico atteso |

`AssetReference` contiene soltanto:

```text
path: AssetPath
logicalSha256: SHA256
```

Il digest atteso è quello definito in F.2 per il relativo tipo di documento, **non l’hash dei byte HTTP**.

Non aggiungo seed, numero di persone, CompanyCount, regole nazionali, origin o configurazioni del generatore.

### Relazione tra le release

La release nazionale identifica la combinazione. Le release dei tre documenti sono indipendenti: possono essere uguali, ma non devono esserlo.

Una nuova release nazionale può riutilizzare un documento base o un catalogo fotografico precedente mediante lo stesso riferimento e digest. Non occorre ricopiarlo o rinominarlo.

Il ruolo `base` ammette soltanto lo schema base v1; analogamente per gli altri due ruoli. Profili e versioni non vengono inferiti dai nomi dei file.

---

# C. Geografia e classificazioni

## C.1 Confronto finale delle due geografie

### Alternativa A — Normalizzata

Una rappresentazione coerente di Agliè sarebbe:

```json
{
  "areas": {
    "r:Piemonte": {
      "type": "region",
      "name": "Piemonte"
    },
    "p:Piemonte:TO": {
      "type": "provincial-unit",
      "code": "TO",
      "parent": "r:Piemonte"
    }
  },
  "places": [
    {
      "id": "001001",
      "name": "Agliè",
      "population": 2584,
      "areaRefs": ["p:Piemonte:TO", "r:Piemonte"],
      "territory": "Piemonte"
    }
  ],
  "postalCodes": {
    "001001": ["10011"]
  }
}
```

Anche questa alternativa potrebbe mantenere autonomo il modello, usando `territory` invece degli ID locali delle aree. Resterebbero però entità, riferimenti, tipi e controllo della gerarchia.

### Alternativa B — Leggera, scelta

```json
{
  "placeCodeSystem": "it-istat-municipality-it-2026a",
  "territorySystem": "sw-it-2026a-region-v1",
  "postalCodeSystem": "it-postal-observations-it-2026a",
  "places": [
    {
      "id": "001001",
      "name": "Agliè",
      "population": 2584,
      "administrative": [
        {
          "system": "it-provincial-uts-it-2026a",
          "code": "TO"
        }
      ],
      "territory": "Piemonte"
    }
  ],
  "postalCodes": {
    "001001": ["10011"]
  }
}
```

La stringa `Piemonte` svolge entrambe le funzioni che ha già nei dati: valore regionale originale e raccordo con la classificazione socioeconomica. Non occorre duplicarla in un’area amministrativa separata.

I valori dell’esempio sono originali; i nomi dei sistemi sono nuovi identificatori proposti. Il CSV non contiene una denominazione estesa per `TO`, che quindi non viene inventata. SYNTHETIC_WORLD_MODELLO_ITALIAN…

| Aspetto | Normalizzata | Leggera |
|---|---|---|
| Strutture aggiuntive | Tipi, aree, genitori e riferimenti | Solo attributi amministrativi tipizzati |
| Relazioni da mantenere | Area–area e località–area | Località–territorio e località–CAP |
| Validazioni | Esistenza delle aree, cicli, riferimenti, eventuali livelli | Unicità dei sistemi locali e coerenza dei codici |
| Adattamento italiano | Risoluzione indiretta degli attributi | Copia diretta verso `placeRow` |
| Beneficio sui dati disponibili | Nessuna informazione originale aggiuntiva | Rappresentazione sufficiente |

**Scelta definitiva: alternativa B.**

### Caso internazionale ipotetico

Supponiamo una località `AB-0017` appartenente a una regione amministrativa e a un distretto, mentre il modello usa una zona statistica diversa.

Con la geografia normalizzata servirebbero due entità amministrative e il relativo collegamento. Con la soluzione scelta:

```json
{
  "id": "AB-0017",
  "name": "Example Town",
  "population": 1200,
  "administrative": [
    {"system": "example-region-v1", "code": "NORTH"},
    {"system": "example-district-v1", "code": "D-7"}
  ],
  "territory": "STAT-ZONE-A"
}
```

Entrambe le alternative rappresentano l’appartenenza. Soltanto la normalizzata rappresenterebbe anche una gerarchia autonoma interrogabile: **questa capacità non è richiesta dai dati attuali**.

I valori dell’esempio sono inventati per il test strutturale e non descrivono statistiche di un paese reale.

## C.2 Identità e risoluzione dei territori

L’identità territoriale è:

\[
(\text{dimensions.territory.semanticId},\ \text{category.code}).
\]

Il documento base dichiara:

```text
geography.territorySystem
places[i].territory
```

La composizione richiede:

```text
base.geography.territorySystem
    == model.dimensions.territory.semanticId
```

e, per ogni località, esattamente una categoria con:

```text
category.code == place.territory
```

Il confronto è testuale esatto. Non si usano nome della città, sigla provinciale, prefix matching, normalizzazione o alias.

Il modello socioeconomico, autonomamente, verifica che la tassonomia territoriale sia non vuota, ordinata e senza codici duplicati e che tutti i suoi vettori abbiano le dimensioni corrette. Non deve sapere quante località appartengano a ogni territorio.

Durante la composizione sono errori:

| Condizione | Esito |
|---|---|
| Sistema territoriale differente | Incompatibilità; World non costruito |
| Codice della località assente nel modello | Riferimento irrisolto; World non costruito |
| Categoria duplicata | Modello rifiutato già autonomamente |
| Codice valido in due sistemi differenti | Nessuna equivalenza automatica |
| Territorio del modello non usato dalle località | Ammesso nello schema comune; il profilo italiano verifica la copertura attesa |

Per l’Italia `place.territory` viene copiato esattamente in `placeRow.region`. L’ordinale deriva dalla tassonomia socioeconomica, non dall’ordine dei comuni.

L’ordine italiano resta:

```text
Abruzzo
Basilicata
Calabria
Campania
Emilia-Romagna
Friuli-Venezia Giulia
Lazio
Liguria
Lombardia
Marche
Molise
Piemonte
Puglia
Sardegna
Sicilia
Toscana
Trentino-Alto Adige/Südtirol
Umbria
Valle d'Aosta/Vallée d'Aoste
Veneto
```

È l’ordine esplicito presente nel Go allegato; `Piemonte` rimane all’ordinale 11. socioeconomic_facts

## C.3 Binding semantici

I seguenti ID sono proposti per il profilo italiano:

| Dimensione | `semanticId` proposto | Significato autorevole |
|---|---|---|
| Genere nominale/fotografico | `sw-l1-gender-v1` | Bucket operativi `female`, `male` |
| Istruzione | `sw-l2-education-v1` | Massimo livello completato |
| Attività | `sw-l2-primary-activity-v1` | Sei stati primari e relativa applicabilità |
| Industria | `sw-l2-industry-v1` | Attività economica primaria |
| Famiglia professionale | `sw-l2-job-family-v1` | Funzione lavorativa primaria |
| Livello professionale | `sw-l2-job-level-v1` | Responsabilità e autorità organizzativa |
| Fascia Company | `sw-l2-company-size-band-v1` | Intervalli del personale modellato |
| Territorio | `sw-it-2026a-region-v1` | Le venti etichette regionali della baseline |

Questi ID non sono una promessa di equivalenza con classificazioni ufficiali esterne.

Un profilo supportato contiene un **binding chiuso** tra ID semantico, ruolo della dimensione, codici ammessi, ordine, associazioni di ramo e soglie applicabili. La validazione confronta queste proprietà, non soltanto la stringa `semanticId`.

Non serve un registry remoto o un framework: per l’Italia sono le classificazioni già note al codice, rese esplicitamente verificabili all’ingresso.

Un ID sconosciuto può appartenere a un documento strutturalmente valido, ma non a un modello eseguibile dal runtime italiano. Le descrizioni facoltative servono al lettore; non fanno approvare una classificazione sconosciuta.

Le tassonomie nominali mantengono un ordine **di indirizzamento**, non un rango sociale o economico. Education e JobLevel conservano il significato ordinale del contratto italiano; SizeBand quello derivato dal conteggio. SYNTHETIC_WORLD_V1_LAYER2_CONTR…

---

# D. Distribuzioni numeriche e assi

## D.1 Registro chiuso dei modelli

Le seguenti sequenze sono parte dello schema v1. Un ordine differente viene rifiutato anche quando il prodotto delle dimensioni coincide.

| Modello | `axes` esatto | `outcome`, quando presente |
|---|---|---|
| `companyIndustry` | `[]` | `industry` |
| `employeeCount` | `["industry"]` | — |
| `education` | `["age.education","territory"]` | `education` |
| `primaryActivity` | `["age.activity","education","territory"]` | `activity` |
| `employeeJobFamily` | `["education","territory"]` | `jobFamily` |
| `employeeJobLevel` | `["age.employeeLevel","education","jobFamily"]` | `jobLevel` |
| `selfEmployedJobFamily` | `["education","territory"]` | `jobFamily` |
| `selfEmployedIndustry` | `["jobFamily"]` | `industry` |
| `employer.industryFactors` | `["jobFamily","industry"]` | — |
| `employer.sizeFactors` | `["jobLevel","positiveSizeBand"]` | — |
| `income.employee` | `["territory","jobFamily","jobLevel","industry","sizeBand"]` | — |
| `income.selfEmployed` | `["age.selfEmployedIncome","territory","jobFamily","industry"]` | — |
| `income.unemployed` | `["age.unemployedIncome","education","territory"]` | — |
| `income.student` | `["age.studentIncome","education","territory"]` | — |
| `income.retired` | `["age.retiredIncome","education","territory"]` | — |
| `income.inactive` | `["age.inactiveIncome","education","territory"]` | — |

`age.x` riferisce precisamente `agePartitions.x`.

Le dimensioni comuni possono cambiare cardinalità e classificazione nazionale. I nomi del registro, i ruoli dei genitori, il numero dei rami e le famiglie numeriche non possono cambiare.

Questa tabella conserva il grafo italiano: per esempio JobLevel non legge direttamente il territorio, Employer non legge direttamente età o istruzione e Income employee non usa età, istruzione, identificatore del datore o EmployeeCount grezzo come assi. SYNTHETIC_WORLD_V1_LAYER2_DETER…

## D.2 Indici e lunghezze

Per assi di cardinalità \(s_0,\ldots,s_k\) e coordinate valide \(d_0,\ldots,d_k\):

\[
\operatorname{offset}
=
\sum_{j=0}^{k}d_j\prod_{h=j+1}^{k}s_h.
\]

È row-major: primo asse più esterno, ultimo più rapido.

Ogni prodotto e ogni somma devono essere controllati prima dell’indicizzazione. Non si usa l’ordine delle mappe.

Per una distribuzione categorica:

\[
\text{len(weights)}
=
\left(\prod \text{cardinalità degli axes}\right)
\times\text{cardinalità dell’outcome}.
\]

Il prodotto vuoto vale 1. La riga condizionata `r` inizia a `r × outcomeWidth`.

Per una matrice di fattori o una `tupleMap`, la lunghezza coincide con il prodotto degli assi. Per `employeeCount`, `len(laws)` coincide con la cardinalità di Industry.

Non viene serializzato `shape`: è derivabile univocamente.

## D.3 Pesi categorici

Ogni riga deve:

- contenere tutti gli slot del risultato, inclusi quelli con peso zero;
- avere totale positivo rappresentabile in `U64`;
- avere MCD dei pesi non nulli uguale a 1.

Una riga deterministica può quindi essere `[0,1,0]`. `[0,2,0]` non è una rappresentazione runtime canonica accettabile.

**Il loader rifiuta, non normalizza.** Non divide per MCD e non elimina categorie. La produzione di nuovi modelli può applicare le regole di canonicalizzazione prima della pubblicazione; la conversione italiana copia i pesi già finali.

Le righe rimangono complete anche per combinazioni con probabilità upstream zero. Questo comportamento è già verificato dal compilatore allegato. socioeconomic_model_compile socioeconomic_model_compile

## D.4 Leggi discrete

Per ogni `DiscreteLaw`:

\[
1\leq\text{len(values)}=\text{len(weights)}.
\]

I supporti sono `U32`, strettamente crescenti e unici. I pesi sono `U32` positivi, con MCD 1 e totale rappresentabile in `U64`.

Zero è un supporto numerico legittimo. Un supporto con peso zero deve essere già assente nella rappresentazione canonica.

Il loader non ordina supporti, non fonde duplicati e non omette punti.

Per EmployeeCount, leggi uguali in posizioni Industry differenti sono ammesse: ogni posizione conserva il proprio significato.

Per Income:

- le leggi sono locali al ramo;
- due leggi canoniche identiche nello stesso ramo sono vietate;
- tutte le leggi devono essere referenziate;
- la prima legge distinta incontrata nella traversata della mappa ha ID 0, la successiva ID 1 e così via.

La verifica procede ignorando i `null` strutturali: al primo incontro di un ID nuovo esso deve coincidere con il prossimo ID atteso. Gli ID già incontrati possono ripetersi. Alla fine devono essere stati incontrati tutti gli ID.

Nessuna deduplicazione attraversa i rami.

## D.5 Partizioni d’età

Per una partizione con limiti \(l_0,\ldots,l_{n-1}\):

```text
l0 == ageDomain.minimum
l0 < l1 < ... < ln-1 <= ageDomain.maximum
```

Il bucket dell’età `a` è:

\[
b(a)=\max\{j:l_j\leq a\}.
\]

La ricostruzione di `compiled` usa tutte le età intere del dominio in ordine crescente:

```text
compiled[a - minimum] = b(a)
```

Gli intervalli sono contigui per costruzione: ciascuno termina prima del limite successivo; l’ultimo termina al massimo del dominio.

Per l’Italia:

| Nuova partizione | Proprietà originale | Limiti |
|---|---|---|
| `education` | `Education` | 21,25,35,45,55,65 |
| `activity` | `PrimaryActivity` | 21,25,35,45,55,60,65,70 |
| `employeeLevel` | `Employee.JobLevel` | 21,25,35,45,55,65 |
| `selfEmployedIncome` | `Income[s]` | 21,25,35,45,55,65 |
| `unemployedIncome` | `Income[u]` | 21,25,35,45,55,65 |
| `studentIncome` | `Income[t]` | 21,25,30,40,55,65 |
| `retiredIncome` | `Income[r]` | 21,45,55,60,65,70 |
| `inactiveIncome` | `Income[i]` | 21,25,35,45,55,65 |

Ho verificato sugli allegati che questa ricostruzione coincide con tutte le otto mappe originali. Il confronto Go futuro deve ripetere la verifica delle 440 posizioni.

## D.6 Income: riferimenti, `null` e zero

Nello schema v1 vale:

```text
income.employee.tupleMap[k] è null
    se e soltanto se
la coordinata sizeBand della tupla è zero.
```

Qui “zero” significa **prima fascia, il cui intervallo è il solo conteggio 0**, non un codice testuale universalmente imposto.

Negli altri cinque rami `null` è vietato. Nel ramo employee è vietato nelle fasce positive.

| Rappresentazione | Significato |
|---|---|
| `tupleMap[k] = null` | Tupla strutturalmente non applicabile |
| `tupleMap[k] = 0` | Prima legge del ramo |
| `values[h] = 0` | Importo monetario zero |
| Campo o elemento mancante | Documento incompleto, da rifiutare |

Per l’Italia:

```text
null ↔ 65535
```

esclusivamente negli 11.200 slot employee della fascia zero. Tutti gli ID validi restano identici.

Il numero 65535 **non è riservato nello schema internazionale**: fuori dal profilo italiano potrebbe essere un ID valido se il ramo contenesse abbastanza leggi. È `null`, non un valore numerico convenzionale, a rappresentare l’inapplicabilità.

Il runtime italiano continua a utilizzare il proprio sentinel interno e i propri limiti. Un accesso operativo a una tupla non applicabile è un errore, non un reddito zero. SYNTHETIC_WORLD_V1_LAYER2_DETER…

## D.7 Companies ed Employer

La famiglia internazionale mantiene:

```text
Company Industry
    → EmployeeCount
    → SizeBand derivata
```

Nessun record Company già generato viene serializzato.

Per `n > 0`, la formula resta:

\[
W=\min(n,C)\times I[F,I]\times S[J,B^+].
\]

Per `n = 0`, `W = 0`.

Cap e fattori sono positivi nei limiti dichiarati. Nessuna normalizzazione o riduzione per MCD viene applicata a fattori o pesi Employer.

Il numero delle condizioni è:

\[
|\text{JobFamily}|\times|\text{JobLevel}|.
\]

Ogni condizione richiede un indice completo, ordinato per CompanyIndex, con totale positivo. Anche le condizioni upstream oggi non raggiungibili devono avere un indice valido.

Il modello autonomo deve avere supporto EmployeeCount positivo in almeno un’Industry con massa positiva. Questo non garantisce che ogni seed produca almeno una Company positiva: il controllo sul roster effettivo resta nella costruzione. Un universo realizzato interamente a zero causa fallimento, senza retry del seed o datore forzato. SYNTHETIC_WORLD_V1_LAYER2_D2A_v…

## D.8 Confine internazionale

Sono rappresentabili nello stesso schema nuove cardinalità, coorti, categorie educative, industrie, famiglie, livelli, territori, soglie Company e unità monetarie compatibili con il dominio numerico.

Richiedono una nuova versione della famiglia: un ramo aggiuntivo con applicabilità diversa, un nuovo genitore statistico, Employer con altra fattorizzazione, Income negativo, distribuzioni continue/interpolate o una nuova variabile latente.

Una tassonomia nazionale diversa può essere rappresentabile ma non eseguibile nel Go italiano. **Validità del documento e supporto del runtime restano due risultati distinti.**

---

# E. Esempi JSON

## E.1 Documento base ridotto

Questo esempio mostra tutti i campi del formato. È un **frammento illustrativo ridotto, non una release italiana valida**: mancano quasi tutti i record e quattro coorti.

I valori delle voci sono originali; gli identificatori di formato, profilo e sistemi sono proposti. `Agostina` e il relativo atomo/peso sono verificati nel CSV allegato. first_names(2)

```json
{
  "schema": "sw-base-data-v1",
  "release": "it-2026a-json1",
  "profile": "it-2026a-base-v1",
  "country": "IT",
  "anchorDate": "2026-01-01",
  "genders": {
    "semanticId": "sw-l1-gender-v1",
    "categories": [
      {"code": "female"},
      {"code": "male"}
    ]
  },
  "ages": [
    {"age": 21, "weight": 609156}
  ],
  "cohorts": [
    {
      "id": "2000-2005",
      "fromYear": 2000,
      "toYear": 2005
    }
  ],
  "firstNames": [
    {
      "gender": "female",
      "cohort": "2000-2005",
      "entries": [
        {
          "display": "Agostina",
          "asciiAtom": "agostina",
          "weight": 1
        }
      ]
    },
    {
      "gender": "male",
      "cohort": "2000-2005",
      "entries": [
        {
          "display": "Vito",
          "asciiAtom": "vito",
          "weight": 1
        }
      ]
    }
  ],
  "lastNames": [
    {
      "display": "Abate",
      "asciiAtom": "abate",
      "weight": 2
    }
  ],
  "geography": {
    "placeCodeSystem": "it-istat-municipality-it-2026a",
    "territorySystem": "sw-it-2026a-region-v1",
    "postalCodeSystem": "it-postal-observations-it-2026a",
    "places": [
      {
        "id": "001001",
        "name": "Agliè",
        "population": 2584,
        "administrative": [
          {
            "system": "it-provincial-uts-it-2026a",
            "code": "TO"
          }
        ],
        "territory": "Piemonte"
      }
    ],
    "postalCodes": {
      "001001": ["10011"]
    }
  },
  "streetNames": {
    "kind": "fragment",
    "entries": [
      {"text": "dell'Unità", "weight": 4}
    ]
  }
}
```

## E.2 Modello socioeconomico: estratto strutturale

È un **frammento illustrativo**, non un documento caricabile: sono omesse dimensioni, partizioni, modelli e parti dei vettori. Gli array mostrati sono prefissi nelle loro posizioni originali, non nuove distribuzioni ridotte.

```json
{
  "schema": "sw-socioeconomic-data-v1",
  "release": "it-2026a-json1",
  "profile": "it-2026a-socioeconomic-v1",
  "country": "IT",
  "anchorDate": "2026-01-01",
  "sourceModel": {
    "schema": "SW-L2-D2B2-logical-v0",
    "byteSha256": "f049103241e6e43ffb5dd4a1247b64b2d0ca1bff787950f25284b5b435fd21a8"
  },
  "ageDomain": {
    "minimum": 21,
    "maximum": 75
  },
  "companyCount": 65536,
  "dimensions": {
    "education": {
      "semanticId": "sw-l2-education-v1",
      "categories": [
        {"code": "below-primary"},
        {"code": "primary"},
        {"code": "lower-secondary"},
        {"code": "upper-secondary"},
        {"code": "post-secondary-non-tertiary"},
        {"code": "tertiary"},
        {"code": "doctoral"}
      ]
    },
    "territory": {
      "semanticId": "sw-it-2026a-region-v1",
      "categories": [
        {"code": "Abruzzo"}
      ]
    }
  },
  "agePartitions": {
    "education": {
      "lowerBounds": [21, 25, 35, 45, 55, 65]
    }
  },
  "incomeUnit": {
    "currency": "EUR",
    "decimalPlaces": 0,
    "period": "year",
    "basis": "gross-personal-monetary-annualized"
  },
  "models": {
    "education": {
      "axes": ["age.education", "territory"],
      "outcome": "education",
      "weights": [
        2735501,
        19122730,
        524395639,
        3161678165,
        97788024,
        487775224,
        1472012
      ]
    },
    "income": {
      "employee": {
        "axes": [
          "territory",
          "jobFamily",
          "jobLevel",
          "industry",
          "sizeBand"
        ],
        "tupleMap": [
          null, 0, 0, 0, 0,
          null, 0, 0, 1, 1
        ],
        "laws": [
          {
            "values": [0, 382, 1144],
            "weights": [8589935, 11162442, 11162442]
          }
        ]
      }
    }
  }
}
```

La riga Education è quella per il primo bucket, 21–24, e Abruzzo. I sette numeri sono originali.

Per Income, l’offset 7 rappresenta:

```text
Abruzzo
production-and-fieldwork
individual-contributor
production-and-utilities
small
```

La sua legge è 0. All’offset 8, fascia `medium`, la legge è 1. Il prefisso della legge 0 conserva il supporto monetario zero; non è una rappresentazione di inapplicabilità.

## E.3 Catalogo fotografico

Questo esempio è una fixture illustrativa di struttura. ID/genere/età dei due candidati derivano dalle informazioni locali del dossier; **seed e release sono valori di esempio**, non quelli di un catalogo remoto verificato.

```json
{
  "schema": "sw-photo-catalog-v1",
  "release": "faces-json1",
  "genderTaxonomy": {
    "semanticId": "sw-l1-gender-v1",
    "categories": [
      {"code": "female"},
      {"code": "male"}
    ]
  },
  "selection": {
    "version": "v1",
    "seed": "0000000000000000000000000000000000000000000000000000000000000001"
  },
  "faces": [
    {
      "id": "face-000013",
      "gender": "female",
      "requestedAge": 21,
      "renditions": [
        {
          "width": 256,
          "height": 256,
          "path": "synthetic-world/photos/v1/256/face-000013.webp"
        }
      ]
    },
    {
      "id": "face-000037",
      "gender": "male",
      "requestedAge": 68,
      "renditions": [
        {
          "width": 256,
          "height": 256,
          "path": "synthetic-world/photos/v1/256/face-000037.webp"
        }
      ]
    }
  ]
}
```

Non aggiungo hash o dimensioni in byte delle immagini perché non sono stati forniti per questi asset.

## E.4 Descrittore

Template JSON: le tre stringhe tra parentesi angolari sono **metavariabili editoriali**, da sostituire con digest reali dei documenti completi. Il template non è un asset pubblicabile.

```json
{
  "schema": "sw-country-release-v1",
  "release": "it-2026a-json1",
  "country": "IT",
  "base": {
    "path": "synthetic-world/countries/IT/it-2026a-json1/base.json",
    "logicalSha256": "<D_BASE>"
  },
  "socioeconomic": {
    "path": "synthetic-world/countries/IT/it-2026a-json1/socioeconomic.json",
    "logicalSha256": "<D_SOCIOECONOMIC>"
  },
  "photos": {
    "path": "synthetic-world/photos/faces-json1/catalog.json",
    "logicalSha256": "<D_PHOTOS>"
  }
}
```

Non assegno digest inventati a documenti convertiti che non esistono ancora.

---

# F. Identità, fingerprint e versioni

## F.1 Canonicalizzazione esatta

La nuova funzione di serializzazione, indicata qui come `C1`, adotta le convenzioni frozen applicabili e aggiunge soltanto il supporto al `null` previsto dal nuovo schema. Il protocollo esistente specifica già ordine delle proprietà, escape, interi e LF finale. SYNTHETIC_WORLD_V1_LAYER2_DETER…

### Decodifica

Prima della canonicalizzazione:

1. Il documento deve essere UTF-8 valido, senza BOM iniziale.
2. Deve contenere un solo oggetto JSON seguito soltanto da whitespace JSON.
3. Commenti e trailing comma sono vietati.
4. Le chiavi duplicate sono rifiutate dopo la decodifica degli escape: `"a"` e `"\u0061"` sono la stessa chiave.
5. Le coppie surrogate correttamente accoppiate negli escape vengono decodificate nel corrispondente valore scalare; surrogate isolate sono rifiutate.
6. Nessun byte invalido viene sostituito con U+FFFD.
7. Si applicano la grammatica intera e i limiti del campo.
8. Non sono ammessi booleani, poiché nessuno dei quattro schemi li prevede.

### Serializzazione

`C1(x)` produce:

| Elemento | Regola esatta |
|---|---|
| Codifica | UTF-8, senza BOM |
| Oggetti | Proprietà ordinate lessicograficamente per sequenza di code point Unicode decodificati; prefisso più corto prima |
| Separatori | `,` e `:`, senza spazi |
| Interi | Decimale minimo, senza segno o zeri iniziali |
| Stringhe | Delimitate da `"` |
| Virgolette/backslash | `\"` e `\\` |
| Controlli con escape breve | `\b`, `\t`, `\n`, `\f`, `\r` |
| Altri U+0000–U+001F | `\u00xx`, esadecimale minuscolo |
| Altri scalari | Letterali UTF-8, inclusi `/`, U+2028 e U+2029 |
| `null` | Esattamente `null` |
| Terminazione | Esattamente un LF, byte `0A` |

Non si effettua HTML escaping. L’ordine non è quello delle unità UTF-16 di un linguaggio ospite.

### Array

Mantengono l’ordine: categorie, età, coorti, gruppi nominali, voci, località, attributi amministrativi, CAP, assi, limiti, supporti, pesi, leggi e mappe.

Si ordinano canonicamente soltanto le due collezioni dichiarate non ordinate nel trasporto fotografico: candidati e rendition. Duplicati non vengono eliminati durante l’ordinamento.

Esempio:

```text
Ingresso:
{"b":"Agli\u00e8", "a":0}

Byte canonici:
{"a":0,"b":"Agliè"}
```

Dopo `}` compare esattamente il byte `0A`.

## F.2 Framing e digest dei documenti

Si riutilizza il framing KFD1 esistente, non una concatenazione di stringhe.

Indico con:

```text
H(domain, fields) = SHA-256(KFD1(domain, fields))
```

I campi sono componenti tipizzati esistenti; i digest passati come bytes sono i **32 byte grezzi**, non le 64 cifre esadecimali.

KFD1 ordina i nomi ASCII dei componenti, ne codifica tipi e lunghezze e distingue stringhe da bytes. Il layout e i limiti restano quelli del protocollo frozen. SYNTHETIC_WORLD_V1_LAYER2_DETER…

Per ogni documento valido e normalizzato:

```text
D = H(domain, { B("document", C1(document)) })
```

| Documento | Dominio esatto |
|---|---|
| Base | `fakedata/base-data/json-v1` |
| Socioeconomico internazionale | `fakedata/socioeconomic-data/json-v1` |
| Catalogo fotografico | `fakedata/photo-catalog/json-v1` |
| Descrittore, quando se ne registra il digest | `fakedata/country-release/json-v1` |

**Scelta esplicita:** questi digest coprono l’intero documento logico, compresi `release`, descrizioni facoltative, percorsi e `sourceModel`, quando presenti.

Questo evita che il descrittore accetti un documento la cui release o provenienza dichiarata è stata modificata lasciando invariato il digest atteso. Non causa perturbazioni indiscriminate della generazione, perché i fingerprint di generazione usano proiezioni separate.

Il digest di un documento non viene materializzato dentro quel documento.

## F.3 Identità distinte

| Identità | Definizione e uso |
|---|---|
| Hash fisico | SHA-256 dei byte effettivi; utile per originali, attestazioni e integrità |
| Digest logico del documento | Formula F.2; indipendente da whitespace e ordine delle proprietà |
| Release | Etichetta immutabile di pubblicazione, non sostituto del contenuto |
| Identità del pacchetto | Combinazione dei tre digest logici e del paese |
| Fingerprint di generazione | Proiezione dei soli input pertinenti al componente |
| Identità della selezione fotografica | Candidati, attributi e discriminanti della selezione |
| Identità del catalogo completo | Comprende anche distribuzione WebP e metadati dichiarati |

L’identità del pacchetto è:

```text
H("fakedata/country-package/v1", {
    S("country", country),
    B("base", D_base),
    B("socioeconomic", D_socioeconomic),
    B("photos", D_photos)
})
```

Non comprende percorsi del descrittore o la sua etichetta `release`. Due descrittori che coordinano esattamente gli stessi tre documenti per lo stesso paese hanno la stessa identità di pacchetto, anche se distribuiti da percorsi diversi.

Non esiste un ciclo: i documenti non contengono il digest del descrittore e il descrittore contiene soltanto digest dei documenti.

## F.4 Sei fingerprint Layer 1

La versione deterministica nuova è identificata come:

```text
layer1-json-v1
```

Per ogni famiglia `f`:

```text
F_f = H(
    "fakedata/layer1-json/v1/" + f,
    {
        S("country", base.country),
        S("profile", base.profile),
        B("data", C1(P_f(base)))
    }
)
```

Le proiezioni sono chiuse:

| Famiglia `f` | Oggetto esatto `P_f(base)` |
|---|---|
| `ages` | `{anchorDate, ages}` |
| `first-names` | `{genders, cohorts, firstNames}` |
| `last-names` | `{lastNames}` |
| `places` | `{placeCodeSystem, territorySystem, places}` |
| `postal-codes` | `{placeCodeSystem, postalCodeSystem, postalCodes}` |
| `street-names` | `{streetNames}` |

I campi indicati vengono copiati con i loro nomi e valori. Non sono inclusi `release`, provenienza, percorsi o digest complessivo del documento.

Per le tassonomie nella proiezione dei nomi propri, si includono `semanticId` e categorie con i codici ordinati; si escludono le sole `definition` esplicative. Questa esclusione è prescritta, non lasciata al decoder.

Il profilo è incluso perché identifica l’interpretazione nazionale delle voci. Cambiare soltanto un dataset dentro lo stesso profilo cambia soltanto il relativo fingerprint. Cambiare il profilo è invece un cambiamento esplicito dell’interpretazione e può cambiare tutte le famiglie.

### Collegamento agli stream correnti

Si conservano i nomi dei sei componenti bytes già presenti, sostituendone il valore soltanto nel nuovo percorso:

| Fingerprint | Componente esistente | Stream interessato |
|---|---|---|
| Età | `age-data-sha256` | `identity/birth/age` |
| Nomi | `first-name-data-sha256` | `identity/name/first` |
| Cognomi | `last-name-data-sha256` | `identity/name/last` |
| Località | `places-data-sha256` | I percorsi correnti di selezione uniforme e pesata del comune |
| CAP | `postal-data-sha256` | `geography/place/postal` |
| Strade | `street-name-data-sha256` | `geography/address/fragment` |

I nomi letterali dei due percorsi di selezione del comune vanno riscontrati in `layer1.go`: non sono riportati integralmente nei sorgenti allegati. La prescrizione è riutilizzare precisamente quei percorsi, non sceglierne di nuovi.

Il dossier conferma la separazione attuale e che genere, giorno di nascita, ramo 95/5, tipo stradale e civico hanno stream propri. Non si aggiungono a questi tutti i fingerprint. SYNTHETIC_WORLD_MODELLO_ITALIAN…

### Isolamento e dipendenze funzionali

Cambiare le età può cambiare nascita, coorte, nome, fotografia e fatti socioeconomici attraverso i loro input effettivi. Non significa che tutti i fingerprint debbano cambiare.

Analogamente, modificare una località può cambiare territorio e CAP applicabile senza cambiare il fingerprint della tabella postale.

Il requisito è isolamento degli indirizzi casuali, **non indipendenza funzionale dei campi**.

## F.5 Fotografie

Definisco la proiezione:

```text
Q = {
    genderTaxonomy: {
        semanticId,
        categories: [{code}, ...]
    },
    selection: {
        version,
        seed
    },
    faces: [
        {id, gender, requestedAge},
        ...
    ]
}
```

I candidati sono nell’ordine normalizzato per ID.

Il fingerprint della selezione è:

```text
F_photo = H(
    "fakedata/photo-selection/json-v1",
    { B("selection", C1(Q)) }
)
```

Il seed viene mantenuto come stringa esadecimale canonica dentro `Q`; il risultato `F_photo` viene passato agli stream come 32 byte grezzi.

Questo rende visibile una modifica di ID, genere, età o insieme dei candidati anche quando `selection.seed` resta invariato.

### Selezione operativa

Per una persona:

1. considerare soltanto i candidati dello stesso codice di genere;
2. trovare la distanza minima tra `requestedAge` e AnchorAge;
3. raccogliere tutti i candidati a quella distanza nell’ordine degli ID;
4. selezionare uniformemente tra essi con lo stream fotografico corrente.

Non c’è una distanza massima, non serve una fotografia per ogni età e non entra ReferenceDate. Il meccanismo è quello documentato nel progetto. SYNTHETIC_WORLD_MODELLO_ITALIAN…

Nel nuovo ingresso:

- `photo/select` mantiene il percorso corrente;
- il discriminante di versione riceve `selection.version`;
- il componente checksum usato per la selezione riceve `F_photo`, non il seed grezzo;
- primitive, regola dei pareggi e indirizzamento per persona restano invariati.

È un cambiamento esplicito della nuova baseline Layer 1, non un cambiamento dei vecchi cataloghi.

### Cosa cambia quale identità

| Modifica | `F_photo` | Digest catalogo |
|---|---:|---:|
| Aggiunta/rimozione di un candidato | Cambia | Cambia |
| ID, genere o età richiesta | Cambia | Cambia |
| Seed o versione di selezione | Cambia | Cambia |
| Ordine fisico dei candidati | Invariato | Invariato dopo normalizzazione |
| Percorso/rendition/hash/byte dichiarati | Invariato | Cambia |
| Release di pubblicazione | Invariato | Cambia |
| Whitespace o escape equivalenti | Invariato | Invariato |

Servono quindi due valori distinti nel catalogo preparato: fingerprint della selezione e digest del contenuto completo.

La nuova baseline usa il primo per la selezione; il secondo per identificare il catalogo dello snapshot e per le informazioni fotografiche già previste dal checkpoint. Non viene aggiunto alcun controllo socioeconomico al checkpoint.

Una modifica non dichiarata dei byte WebP allo stesso percorso non è rilevabile dal solo catalogo. Non attribuisco al formato una garanzia che richiede immutabilità o verifica dei byte.

## F.6 Modello socioeconomico frozen

Sia `L` il modello originale e `P(M)` la proiezione del nuovo modello italiano.

Le proprietà da verificare sono separate:

\[
P(M)=L
\]

come valori logici, quindi:

\[
C(P(M))=C(L)
\]

secondo la canonicalizzazione **frozen**, e infine:

\[
M_{\text{all}}=\operatorname{SHA256}(C(L)).
\]

Il digest internazionale `D_socioeconomic` è diverso: comprende il nuovo contenitore, le dimensioni, l’unità e la release.

Per l’allegato ho verificato che:

```text
byte originali == C(L) secondo la convenzione documentata
SHA-256 = f049103241e6e43ffb5dd4a1247b64b2d0ca1bff787950f25284b5b435fd21a8
```

**La verifica di `canonicalModel()` Go resta necessaria.** La coincidenza non viene dedotta dal nome del file.

I quindici fingerprint frozen continuano a usare i loro sottoalberi e le loro mappe d’età originali. Non ricevono tassonomie del nuovo trasporto, digest del pacchetto, release internazionali o percorsi.

Per riutilizzarli servono anche equivalenza dei significati, ordinali, soglie, unità e ancoraggio: l’uguaglianza dei soli numeri non basta.

## F.7 Versioni del World e ripresa

Per il nuovo ingresso propongo:

```text
PersonSpecVersion: person-v2
Protocollo dati Layer 1: layer1-json-v1
Policy applicativa: generation-v5
WorldModelVersion della radice: world-v1, invariata
```

La radice casuale effettiva rimane quella del World base. Non si aggiungono digest del pacchetto o seed socioeconomici. Il protocollo L2 prescrive infatti il riuso della factory effettiva e della radice esistente. SYNTHETIC_WORLD_V1_LAYER2_DETER…

Il WorldID base continua a escludere foto e dati socioeconomici. Usa la release del documento base, non la release nazionale coordinata: altrimenti una sostituzione del solo catalogo fotografico potrebbe cambiare indebitamente l’identità socioeconomica.

La compatibilità del nucleo L2 non autorizza automaticamente il nuovo World negli attuali controlli di provenienza. L’ammissione della nuova baseline e il caricamento internazionale devono essere disciplinati dal **nuovo contratto di integrazione**, lasciando intatti protocollo e documenti frozen.

`generation-v5` è l'unica policy supportata per nuove preparazioni non vuote; `empty-v1` resta supportato per i sistemi vuoti. Non si creano né si riprendono preparazioni v3/v4. Le policy e i checkpoint v3/v4 non vengono convertiti o reinterpretati come v5, neppure quando gli indici sono ancora validi.

I sistemi già Available conservano i record persistiti, senza rigenerazione o cancellazione automatica; restano le operazioni applicative indipendenti dal generatore. Le preparazioni legacy incomplete devono seguire il percorso permanente del worker verso Failed, con motivazione esplicita di policy non più supportata, secondo il nuovo contratto di integrazione. Failed e Deleting restano gestibili secondo le regole applicative esistenti.

---

# G. Validazioni

## G.1 Livello A — Documento autonomo

| Documento | Verifiche |
|---|---|
| Tutti | Sintassi, UTF-8, unico documento, chiavi duplicate, campi chiusi, tipi, limiti numerici, `null`, identificatori |
| Base | Età ordinate, coorti non sovrapposte, riferimenti ai gruppi, copertura cartesiana, pesi, località uniche, copertura postale |
| Socioeconomico | Tassonomie interne, assi prescritti, dimensioni, partizioni, canonicalità numerica, leggi e riferimenti |
| Foto | ID e rendition unici, generi dichiarati, percorsi, seed e integrità dichiarata |
| Descrittore | Ruoli obbligatori, schema, paese, percorsi e digest formalmente validi |

Un fallimento rifiuta quel candidato di acquisizione. Non viene restituito un oggetto parzialmente valido.

Il socioeconomico supera questo livello senza leggere il documento base.

## G.2 Livello B — Profilo italiano

Il profilo base verifica:

| Area | Vincoli italiani |
|---|---|
| Paese e tempo | IT, ancoraggio 2026-01-01 |
| Genere | `female`, `male`, binding e ordine previsti |
| Età | 55 valori consecutivi 21–75 |
| Coorti | Cinque ID e intervalli originali |
| Nomi | Dieci gruppi, 80 voci ciascuno, quote 20/40/20 per pesi 4/2/1 |
| Cognomi | 600 voci, quote 120/240/240 per pesi 4/2/1 |
| Strade | 400 frammenti, quote 80/160/160 per pesi 4/2/1 |
| Geografia | 7.896 comuni, codici e ordine originali, sigle provinciali/UTS valide, popolazioni positive |
| CAP | Copertura completa e liste italiane valide |
| Testi | Regole D1-B e rettifica; atomi conservati e compatibili con il runtime |

I controlli curatorali — attestazioni umane, ranking, fonti e ricostruzione dell’atomo — restano verifiche offline della release. Il caricamento non ricostruisce la curation.

La differenza tra regole D1-B e parser corrente va mantenuta esplicita: il contratto ammette un repertorio ASCII più generale di quello accettato dal piccolo validatore allegato. La conversione dei dati esistenti non richiede di ampliare quel repertorio né di rigenerare gli atomi. layer1_data

Il profilo socioeconomico verifica:

- tassonomie e ordinali italiani completi;
- ancoraggio e dominio 21–75;
- otto partizioni originali;
- CompanyCount 65.536;
- fasce 0, 1–9, 10–49, 50–249, 250+;
- EUR interi e significato Income frozen;
- cap 4096, forme dei fattori e limiti italiani;
- cardinalità e rappresentazioni supportate dal compilatore;
- proiettabilità completa in `logical-v0`.

La validazione numerica non sostituisce il confronto completo della release con gli originali: quel confronto è un criterio di pubblicazione della migrazione.

## G.3 Livello C — Composizione del World

| Relazione | Controllo |
|---|---|
| Descrittore → documenti | Digest attesi, schemi ammessi, paese e profili supportati |
| Base → socioeconomico | Stesso ancoraggio; tutte le età generabili nel dominio delle otto partizioni |
| Località → territorio | Stesso sistema e risoluzione unica del codice |
| Nascita → coorti | Ogni anno generabile coperto e gruppo disponibile |
| Persona → foto | Tassonomia compatibile e almeno un candidato per ogni genere generabile |
| Profilo foto italiano | Rendition 256×256 disponibile per ogni candidato |
| Modello → runtime | Assi, cardinalità, unità e tipi accettati |
| Company realizzate → Employer | Tutti gli indici richiesti con totale positivo |

Non serve una corrispondenza esatta tra ogni età della persona e un’età fotografica: il selettore usa il candidato più vicino.

I controlli tra documenti possono essere eseguiti durante la preparazione coordinata, prima di istanziare il World. I controlli dipendenti dalle Company effettivamente generate rimangono nella costruzione.

Un errore di composizione non rende invalido un modello autonomo già acquisito; impedisce l’uso di quella combinazione.

---

# H. Mappatura italiana aggiornata

Le corrispondenze numeriche della proposta precedente restano confermate. Le modifiche effettive riguardano geografia, provenienza e identità fotografica.

## H.1 Tutte le colonne dei CSV

| Origine | Destinazione consolidata | Inversa |
|---|---|---|
| `age_weights.age` | `ages[].age` | Copia |
| `age_weights.weight` | `ages[].weight` | Copia |
| `first_names.gender` | `firstNames[].gender` | Ripetere il valore del gruppo |
| `first_names.cohort` | `firstNames[].cohort` | Ripetere l’ID originale |
| `first_names.name` | `entries[].display` | Copia esatta |
| `first_names.ascii_atom` | `entries[].asciiAtom` | Copia esatta |
| `first_names.weight` | `entries[].weight` | Copia |
| `last_names.name` | `lastNames[].display` | Copia esatta |
| `last_names.ascii_atom` | `lastNames[].asciiAtom` | Copia esatta |
| `last_names.weight` | `lastNames[].weight` | Copia |
| `places.istat_code` | `geography.places[].id` | Copia stringa |
| `places.city` | `geography.places[].name` | Copia esatta |
| `places.state_prov` | Attributo con `system=it-provincial-uts-it-2026a`, campo `code` | Risoluzione dell’unico attributo richiesto |
| `places.region` | `geography.places[].territory` | Copia esatta |
| `places.population` | `geography.places[].population` | Copia |
| `postal_codes.istat_code` | Chiave di `geography.postalCodes` | Copia stringa |
| `postal_codes.postal_codes` | Array associato alla chiave | Concatenazione con `;`, senza riordinamento |
| `street_names.name` | `streetNames.entries[].text` | Copia esatta |
| `street_names.weight` | `streetNames.entries[].weight` | Copia |

I metadati aggiunti non alterano le colonne originali. Per tutte queste righe la perdita logica prevista è **nessuna**.

La ricostruzione dei nomi concatena i gruppi nell’ordine originario; quella del CSV postale segue l’ordine dei comuni. Gli originali restano conservati, anche quando si verifica l’equivalenza a livello logico.

## H.2 Tutte le strutture socioeconomiche

| Origine | Destinazione | Proiezione inversa |
|---|---|---|
| `/schema` | `sourceModel.schema` | Ripristino del literal originale |
| `/companyCount` | `companyCount` | Copia |
| `/ages/<x>/lowerBounds` | `agePartitions/<y>/lowerBounds` | Rinomina secondo D.5 |
| `/ages/<x>/compiled` | Derivato | Ricostruzione delle 55 posizioni |
| `/companyIndustry/weights` | `models.companyIndustry.weights` | Array identico |
| `/employeeCount/laws` | `models.employeeCount.laws` | Stesso ordine per Industry |
| Ogni `employeeCount` `values/weights` | Stessi campi della stessa legge | Copia integrale |
| `/education/weights` | `models.education.weights` | Array identico |
| `/primaryActivity/weights` | `models.primaryActivity.weights` | Array identico |
| `/employeeJobFamily/weights` | `models.employeeJobFamily.weights` | Array identico |
| `/employeeJobLevel/weights` | `models.employeeJobLevel.weights` | Array identico |
| `/selfEmployedJobFamily/weights` | `models.selfEmployedJobFamily.weights` | Array identico |
| `/selfEmployedIndustry/weights` | `models.selfEmployedIndustry.weights` | Array identico |
| `/employer/cap` | `models.employer.workforceCap` | Copia |
| `/employer/industryFactors` | `models.employer.industryFactors.values` | Array identico |
| `/employer/sizeFactors` | `models.employer.sizeFactors.values` | Array identico |
| `/income/e` | `models.income.employee` | Rinomina |
| `/income/s` | `models.income.selfEmployed` | Rinomina |
| `/income/u` | `models.income.unemployed` | Rinomina |
| `/income/t` | `models.income.student` | Rinomina |
| `/income/r` | `models.income.retired` | Rinomina |
| `/income/i` | `models.income.inactive` | Rinomina |
| Ogni `/tupleMap` | Stesso campo del ramo | Copia degli ID; `null→65535` nei soli slot previsti |
| Ogni `/laws` | Stesso campo del ramo | Stesso ordine |
| Ogni legge `/values` | Stesso campo | Supporti identici |
| Ogni legge `/weights` | Stesso campo | Masse identiche |

I metadati nuovi vengono verificati prima della proiezione; non vengono inseriti nel vecchio JSON logico.

La proiezione recupera quindi tutte le tredici proprietà principali, non soltanto i modelli numerici più visibili.

## H.3 Catalogo precedente

| Origine | Nuova posizione/trattamento |
|---|---|
| `format=1` | Formato di origine registrato nella migrazione; non confuso con il nuovo `schema` |
| `release` | Copiato in `selection.version` come scelta esplicita del nuovo discriminante, distinta dal precedente discriminante runtime `fakefacegen-v1`. |
| `selectionSeed` | Copiato esattamente in `selection.seed` |
| `faces[].id/gender/requestedAge` | Copia |
| Percorso 256 derivato dal codice | Reso esplicito in `renditions[].path` |
| Dimensioni 256×256 | Rese esplicite |
| Hash/byte non presenti nell’origine | Non inventati |
| Metadati offline del produttore | Conservati nei documenti originali, non aggiunti al catalogo operativo |

Il nuovo fingerprint fotografico cambia intenzionalmente la baseline anche quando i valori sorgente sono conservati. Conservazione dei dati e conservazione dell’indirizzo casuale precedente non sono lo stesso requisito.

---

# I. Integrazione minima con Go ed External Assets

## I.1 Descrittore contro percorsi fissi

Percorsi fissi immutabili sarebbero sufficienti se tutti gli asset evolvessero sempre insieme. Con un catalogo condivisibile, però, la loro associazione dovrebbe essere fissata nel codice o in convenzioni di naming.

Il descrittore aggiunge soltanto tre riferimenti e tre digest e rende esplicita tale associazione. Non sostituisce il resolver e non diventa una configurazione nazionale.

La nuova preparazione risolve un percorso di descrittore fissato dalla policy applicativa, per esempio:

```text
synthetic-world/countries/IT/it-2026a-json1/release.json
```

Non viene introdotta una nuova variabile d’ambiente per scegliere liberamente modelli o release.

## I.2 Sequenza di acquisizione

```text
Policy applicativa versionata
    → descrittore
    → tre documenti tramite lo stesso resolver
    → validazioni autonome e profili
    → confronto dei digest attesi
    → validazioni incrociate
    → strutture immutabili preparate
    → costruzione locale del World
```

Il digest atteso va verificato **dentro il consumo del candidato della singola origin**, così che un contenuto non corrispondente possa fallire e consentire il normale fallback.

Un asset mancante o errato su tutte le origin fa fallire la preparazione. Non si usa una vecchia cache incompatibile, un dataset embedded alternativo o un altro paese.

Un candidato non viene pubblicato nella cache prima dei controlli finali del resolver. Successi riusabili, coordinamento delle chiamate concorrenti e retry dopo errore seguono l’ownership già presente. SYNTHETIC_WORLD_V1_LAYER2_EXTER…

La cache non può essere identificata soltanto dal percorso quando esistono digest attesi differenti. Una chiave equivalente deve comprendere almeno ruolo/schema, percorso e digest atteso, nell’ambito delle origin possedute dal source.

## I.3 Confini del codice

| Componente | Modifica necessaria |
|---|---|
| Source applicativo | Coordinare descrittore e asset con proprietà di successo/retry |
| Decoder base | Decodifica chiusa e preparazione dei dati italiani |
| `italyData` | Riutilizzo di slice, mappe, pesi e bucket; nuovi valori dei sei componenti |
| World base | Unico ingresso operativo italiano con dati JSON già acquisiti, validati e preparati |
| Decoder socioeconomico | Nuovo percorso separato da `logical-v0` |
| Adattatore italiano | Verifica dei metadati e ricostruzione dell’ingresso numerico esistente |
| Compilatore L2 | Riutilizzo per canonicalità, cumulative, leggi e identità |
| Catalogo foto | Distinguere fingerprint di selezione e digest completo |
| Preparazione Simulated System | Sola policy non vuota `generation-v5`, combinazione coerente e ritiro della ripresa v3/v4 |
| Costruttore socioeconomico | Verificare e autorizzare esplicitamente la provenienza della nuova baseline base |

Non si serializzano CSV temporanei e non si riserializza il vecchio JSON per riparsarlo. L’adattamento trasferisce valori tipizzati.

I sei CSV sono input offline della conversione senza perdita di H.1; il runtime finale acquisisce i dati base esclusivamente dal documento JSON internazionale tramite External Assets. Non incorpora i CSV con `go:embed`, non li carica come alternativa e non prevede fallback ai dati embedded. La rimozione fisica degli originali di provenienza non è decisa da questo contratto.

Si riutilizzano strutture runtime, algoritmi italiani e primitive deterministiche esistenti. Restano invariati campionatori, DAG, Company generation, formula Employer, algoritmi di Source Simulation e Oracle e struttura dei sei fatti socioeconomici applicativi. I consumatori ancora necessari del World devono ricevere la nuova base preparata, senza una seconda acquisizione CSV; il binding Source è disciplinato dall'integrazione, non da sostituzioni globali delle costanti.

Il percorso e lo schema del loader socioeconomico storico restano documentati dall’amendment frozen, senza obbligo di mantenerlo nel runtime esclusivamente per generare sistemi v4. Può essere ritirato se non necessario ad altre funzionalità, preservando `SocioeconomicModel`, compilatore e modello numerico. Il nuovo percorso richiede una specifica di integrazione successiva, non un’estensione silenziosa di quel contratto; asset frozen ed evidenze storiche restano intatti. SYNTHETIC_WORLD_V1_LAYER2_EXTER… SYNTHETIC_WORLD_V1_LAYER2_EXTER…

## I.4 Distribuzione delle immagini

Il nuovo JSON non aggiunge fallback WebP.

I metadati usano il resolver; gli URL immagine continuano a usare la prima origin configurata e il percorso dichiarato. La pubblicazione deve quindi garantire lì tutte le rendition richieste.

Questa asimmetria è esplicita e rimane separata dal progetto dei formati. SYNTHETIC_WORLD_EXTERNAL_ASSETS…

## I.5 Ripresa

L’identità del pacchetto non viene aggiunta ai checkpoint come controllo socioeconomico mascherato.

La coerenza tra processi continua a dipendere dall’immutabilità della release e dei riferimenti. Il descrittore rileva asset non corrispondenti ai digest che dichiara; **non rileva da solo una sostituzione coordinata e non autorizzata del descrittore e degli asset tra due processi**.

I checkpoint fotografici esistenti possono identificare il nuovo catalogo secondo la policy v5. Non vengono aggiunti `SocioeconomicWorldID`, `M_all` o digest del modello.

---

# J. Piano di verifica e criteri di accettazione

| Fase | Verifica richiesta | Criterio di accettazione |
|---|---|---|
| 1. Review sul checkout | Contratti, sorgenti, versioni e nuovi confini di ingresso | Nessuna modifica implicita di semantiche frozen |
| 2. Conversione offline | Sei CSV e modello originale come input attestati | Nessuna ricalibrazione, normalizzazione delle masse o conversione monetaria |
| 3. Equivalenza base | Tutte le tuple, stringhe, associazioni e ordini | Ricostruzione integrale dei sei contenuti originali |
| 4. Equivalenza L2 | `P(M)=L` su ogni proprietà | Uguaglianza strutturale completa |
| 5. Identità L2 | Canonicalizzazione Go, digest globale e quindici fingerprint | Identità uguali tra originale e proiezione |
| 6. Strutture runtime | Slice, mappe, pesi, bucket, leggi e cumulative | Stessi valori e relazioni, esclusi i nuovi fingerprint Layer 1 |
| 7. Canonicalizzazione nuova | Escape, Unicode, chiavi duplicate, interi e ordine | Byte canonici e digest riproducibili |
| 8. Acquisizione | Origin fallite, digest errato, retry e concorrenza | Nessun candidato parziale o fallback semantico |
| 9. Nuovo determinismo | Accesso diretto, ordine inverso, interleaving, batch e riavvio | Stessi output a parità degli input pertinenti |
| 10. Fotografie | Candidati e distribuzione modificati separatamente | Perturbazioni delle identità conformi a F.5 |
| 11. Integrazione | Materializzazione, duplicati, omissioni, Income zero e replay | Documenti canonici coerenti e ripresa v5 riproducibile |
| 12. Ritiro legacy | Binario e caricamento base; creazione e stati dei sistemi storici | Nessun CSV embedded o fallback; sole nuove policy v5/empty; Preparing v3/v4 verso Failed senza conversione; Available preservati; gestione Failed/Deleting indipendente dai generatori ritirati |

Il confronto L2 deve coprire espressamente:

```text
12.194 pesi categorici
14 leggi EmployeeCount, con 128 punti ciascuna nella baseline
140 fattori Industry e 16 fattori dimensionali
76.160 riferimenti Income
11.200 sentinel employee
60 leggi Income, con 385 punti ciascuna nella baseline
8 × 55 valori compiled ricostruiti
```

I confronti non devono limitarsi a somme, quantili o campioni.

I test di canonicalizzazione devono includere almeno chiavi equivalenti mediante escape, ordine Unicode non UTF-16, stringhe non ASCII, slash, caratteri di escape, omissione contro `null`, numeri vietati e permutazioni delle collezioni fotografiche.

I test di isolamento devono distinguere due affermazioni: il fingerprint estraneo rimane uguale; il campo discendente può cambiare attraverso un genitore consentito.

I nuovi golden appartengono alla nuova baseline. Non devono riprodurre gli utenti generati dai vecchi CSV.

---

# K. Decisioni davvero residue

## Bloccanti per approvare gli schemi

**Non rimane una scelta architetturale irrisolta nei quattro formati.** Nomi, strutture, riferimenti, canonicalizzazione, proiezioni e comportamento degli errori sono definiti nella presente candidatura.

Rimane l’approvazione tecnica della candidatura, non la scelta futura tra geografie, famiglie numeriche o meccanismi di identità.

## Da verificare sul codice prima dell’implementazione

La singola review Codex deve riscontrare soprattutto tre punti.

**Ammissione della nuova baseline.** Verificare il nuovo ingresso del World, il relativo binding nel wrapper socioeconomico e l'adattamento dei consumatori ancora necessari al ritiro dei caricatori e della generazione v3/v4, senza cambiare la radice frozen o reinterpretare i checkpoint storici.

**Confini fotografici.** Verificare dove il codice usa oggi un unico checksum per selezione, snapshot e checkpoint, separando i due valori soltanto nel percorso nuovo.

**Riutilizzo dei decoder e validatori.** Verificare i nomi effettivi degli stream geografici, le primitive JSON disponibili, i controlli sulle chiavi duplicate e il trattamento delle regole Unicode D1-B senza introdurre trasformazioni implicite o lavoro per persona.

Questi controlli possono determinare la collocazione delle modifiche, non devono lasciare a Codex la scelta di nuove semantiche.

## Rinviabili al deployment o a evoluzioni future

Il catalogo fotografico effettivamente pubblicato, il suo seed e gli eventuali hash WebP restano da acquisire e verificare. Le origin devono rispettare disponibilità e immutabilità della nuova release.

La riproducibilità del producer D2-B2 originario resta distinta dalla conversione fedele del JSON disponibile.

Un secondo paese richiederà il proprio profilo e un consumatore Go compatibile. Un modello fuori dalla famiglia definita richiederà una versione successiva, non proprietà arbitrarie nel JSON.

**La candidatura è quindi pronta per una review tecnica mirata sul repository: geografia leggera, quattro contratti chiusi, proiezione italiana reversibile, sei fingerprint Layer 1 separati e protocollo socioeconomico riutilizzato senza modificarne il contenuto numerico o le identità di nodo.**