| DatasetStrategyEvidenceExact sizeWeight quotas `4 / 2 / 1`Verdict |                               |                                                                                                   |                                              |                             |                       |
| ----------------------------------------------------------------- | ----------------------------- | ------------------------------------------------------------------------------------------------- | -------------------------------------------- | --------------------------- | --------------------- |
| First names                                                       | **EVIDENCE-INFORMED CURATED** | ISTAT per il segmento recente; curation storica esplicita e review intergenerazionale             | **800 righe: 80 per ciascuno dei 10 gruppi** | **20 / 40 / 20 per gruppo** | **READY — contratto** |
| Last names                                                        | **ORIGINAL CURATED**          | Selezione editoriale originale; riferimenti onomastici come controllo, non come lista da estrarre | **600**                                      | **120 / 240 / 240**         | **READY — contratto** |
| Street fragments                                                  | **ORIGINAL CURATED**          | Lexicon originale informato dalla toponomastica italiana; verifica umana delle sei combinazioni   | **400**                                      | **80 / 160 / 160**          | **READY — contratto** |

**ASCII ATOM STRATEGY:** Unicode **16.0.0**, decomposizione **NFD**, conversione delle sole lettere ASCII maiuscole, rimozione controllata dei diacritici, separatori espliciti, **rifiuto dei caratteri non supportati**, collisioni **allowed + reported**.

# L1-D1-B — Italian Curated Plausibility Data Contract

**Market:** `italy`
**MarketDataVersion:** `it-2026a`
**Data della verifica delle fonti:** 15 settembre 2026.

Questo documento congela il contratto di curation, serializzazione e verifica. Non contiene i CSV completi, non implementa il builder e non avvia D1-C, conformemente al perimetro richiesto.

**READY significa che le decisioni necessarie alla curation sono definite. Non significa che i dataset siano già stati costruiti, approvati o verificati sui loro byte finali.**

Le prescrizioni delle sezioni C–G sono **CURATION DECISION**, salvo diversa marcatura. `DEVE`, `NON DEVE` e i numeri indicati sono normativi.

## A. Decisioni e chiusura dei gate

| GateDecisione               |                                                                                                                                                                                  |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **A — First-name evidence** | La serie ISTAT parte dal 1999. Supporta direttamente tutti gli anni del 2000–2005, ma soltanto il 1999 della coorte precedente. Non viene retroproiettata sulle coorti storiche. |
| **B — Historical gap**      | Per le prime quattro coorti si adotta curation documentata, con due valutazioni umane indipendenti. Il 1999 è un riscontro parziale, non una stima del decennio.                 |
| **C — Sizes and weights**   | Si congelano le cardinalità e le quote della tabella iniziale. Nessun ampliamento automatico delle liste.                                                                        |
| **D — Surnames**            | Si sceglie **ORIGINAL CURATED**, senza frequenze nazionali dichiarate né distribuzioni regionali.                                                                                |
| **E — Streets**             | Si sceglie **ORIGINAL CURATED**. La disponibilità di ANNCSU non giustifica, per 400 frammenti, una derivazione da un archivio nazionale di indirizzi.                            |
| **F — ASCII**               | Si sceglie una trasformazione limitata e verificabile, non una translitterazione universale. Nessun comportamento dipendente dalla locale.                                       |

La copertura temporale indicata per il Gate A è documentata dal prodotto ISTAT; i suoi limiti non sono una scelta curatoriale. ([Istat](https://www.istat.it/dati/calcolatori/contanomi/ "https://www.istat.it/dati/calcolatori/contanomi/"))

---

# B. Source evidence

## B.1 ISTAT — Conta nomi e Demo, Nascite

**VERIFIED FACT**

**Product/source:** Conta nomi; sezione Nascite di Demo.
**Publisher:** Istituto nazionale di statistica.
**Locator:** `https://www.istat.it/dati/calcolatori/contanomi/` e `https://demo.istat.it/app/?i=FE3&l=it`.

**Coverage:** nati iscritti in anagrafe, popolazione residente in Italia. **Years:** dal **1999**; l’ultimo anno verificato nel portale Nascite è **2024**. **Sex dimension:** presente. **Frequency information:** conteggi annuali per nome e sesso, con graduatorie annuali. ([Istat Demo](https://demo.istat.it/ "https://demo.istat.it/"))

**Acquisition:** interfaccia pubblica, valori nei grafici e possibilità di esportare i grafici. Non ho verificato un’API batch documentata né un download completo e stabile dell’intero repertorio. ISTAT avverte inoltre che non corregge sistematicamente errori di scrittura, trascrizione o incompatibilità nome–sesso. ([Istat](https://www.istat.it/dati/calcolatori/contanomi/ "https://www.istat.it/dati/calcolatori/contanomi/"))

**License:** le note legali correnti indicano **CC BY 4.0**, salvo diversa indicazione. **Redistributability:** consentita, anche per elaborazioni, rispettando attribuzione, riferimento alla licenza, indicazione delle modifiche e assenza di endorsement implicito. ([Istat](https://www.istat.it/note-legali/ "https://www.istat.it/note-legali/"))

**Role in D1-B:** input empirico per `2000-2005`; corroborazione limitata al 1999 per `1990-1999`.

**ACCEPTABLE LIMITATION:** i nati della popolazione residente sono il riferimento operativo scelto per la plausibilità del repertorio; non vengono presentati come un censimento perfettamente coincidente con tutte e sole le persone fisicamente nate sul territorio italiano.

## B.2 Treccani — Antroponimi e Cognomi

**VERIFIED FACT**

**Product/source:** *Antroponimi* e *Cognomi*, di Carla Marcato, *Enciclopedia dell’Italiano*, 2010.
**Publisher:** Istituto della Enciclopedia Italiana.
**Locator:** le due voci nei riferimenti citati.

**Coverage:** struttura del repertorio onomastico, varianti grafiche, composti, cambiamenti storici e forme cognominali. **Years:** trattazione storica, non una serie annuale omogenea. **Sex dimension:** qualitativa. **Frequency information:** riferimenti e dati storici eterogenei, non una tabella utilizzabile per le cinque coorti richieste. **Acquisition:** consultazione HTML. Le pagine riportano riproduzione riservata. ([Treccani](https://www.treccani.it/enciclopedia/antroponimi_%28Enciclopedia-dell%27Italiano%29/ "https://www.treccani.it/enciclopedia/antroponimi_%28Enciclopedia-dell%27Italiano%29/"))

**CURATION DECISION**

**License/redistributability:** nessuna licenza open verificata per estrarre sistematicamente il repertorio. Si conservano URL e note originali di lettura, non copie degli articoli o delle loro liste.

**Role in D1-B:** corroborazione storica, linguistica e ortografica.

**Limitations:** le graduatorie riconducibili a elenchi telefonici, presenti nella trattazione, sono **escluse** dagli input D1-B. Non si trasformano esempi enciclopedici in frequenze nazionali contemporanee. ([Treccani](https://www.treccani.it/enciclopedia/antroponimi_%28Enciclopedia-dell%27Italiano%29/ "https://www.treccani.it/enciclopedia/antroponimi_%28Enciclopedia-dell%27Italiano%29/"))

## B.3 Comune di Modena — Cognomi con frequenza maggiore dal 2012 al 2022

**VERIFIED FACT**

**Publisher:** Comune di Modena.
**Locator:** `https://opendata.comune.modena.it/dataset/cognomi-con-frequenza-maggiore-dal-2012-al-2022`.

**Coverage:** primi 50 cognomi più frequenti nel Comune, per gli anni **2012–2022**. **Sex dimension:** non indicata. **Frequency information:** repertorio locale di prevalenza. **Acquisition:** il catalogo pubblicizza una risorsa CSV; il relativo download non è stato riprodotto con successo in questa verifica. La scheda dichiara CC BY 4.0, ma il collegamento associato alla licenza punta a un dominio Open Data Commons: un’incoerenza da chiarire prima di un’estrazione. ([Open Data Modena](https://opendata.comune.modena.it/dataset/cognomi-con-frequenza-maggiore-dal-2012-al-2022 "https://opendata.comune.modena.it/dataset/cognomi-con-frequenza-maggiore-dal-2012-al-2022"))

**CURATION DECISION**

**Redistributability:** non assunta sulla sola etichetta del catalogo.

**Role in D1-B:** candidata **scartata come input empirico**. Dimostra la disponibilità di aggregati pubblici locali, ma non fornisce la base nazionale richiesta.

**Limitations:** copertura locale, repertorio corto e acquisizione/licenza non completamente risolte. Non viene resa una dipendenza della curation.

## B.4 ANNCSU — Stradari e consultazione

**VERIFIED FACT**

**Product/source:** Archivio nazionale dei numeri civici delle strade urbane.
**Publisher:** Agenzia delle Entrate e ISTAT.
**Locator:** `https://www.anncsu.gov.it/it/consultazione-dellarchivio/open-data/`.

**Coverage:** archivio nazionale alimentato dai Comuni; non si assume automaticamente la completezza di ogni Comune. **Years:** snapshot correnti, non serie per coorte. **Sex dimension:** non applicabile. **Frequency information:** eventuali ricorrenze delle denominazioni sono conteggi di elementi dell’archivio, non pesi direttamente appropriati per il generatore.

**Acquisition:** stradari e indirizzari nazionali/regionali, CSV compressi ZIP, aggiornati mensilmente, con data nel nome del file. I collegamenti al download massivo sono pubblicati. ([ANNCSU](https://www.anncsu.gov.it/it/consultazione-dellarchivio/open-data/ "https://www.anncsu.gov.it/it/consultazione-dellarchivio/open-data/"))

**License:** la pagina del servizio di consultazione dichiara **CC BY 4.0**. La verifica non viene estesa automaticamente a ogni specifico pacchetto non acquisito. ([ANNCSU](https://www.anncsu.gov.it/it/consultazione-dellarchivio/ricerca-strade-e-numeri-civici/ "https://www.anncsu.gov.it/it/consultazione-dellarchivio/ricerca-strade-e-numeri-civici/"))

**CURATION DECISION**

**Role in D1-B:** riferimento istituzionale per il confronto con la toponomastica reale; **nessuna estrazione di un corpus di denominazioni**.

**Redistributability/limitations:** una derivazione sarebbe una strada possibile da verificare sul pacchetto scelto, ma non è la strategia adottata. Non esiste quindi una dipendenza di licenza ANNCSU per il lexicon originale.

## B.5 OpenStreetMap

**VERIFIED FACT**

**Product/source:** database OpenStreetMap.
**Publisher:** contributori OpenStreetMap; licenza pubblicata dalla OpenStreetMap Foundation.
**Locator:** `https://www.openstreetmap.org/copyright`.

**Coverage:** dati cartografici globali. **Years:** archivio aggiornato, non serie demografica. **Sex dimension:** non applicabile. **Frequency information:** occorrenze cartografiche, non frequenze sintetiche normative. **Acquisition in D1-B:** nessuna.

**License/redistributability:** ODbL, con attribuzione e condizioni applicabili alla redistribuzione di database derivati. ([OpenStreetMap](https://www.openstreetmap.org/copyright "https://www.openstreetmap.org/copyright"))

**CURATION DECISION:** importazione esclusa. Per 400 frammenti, gestire una derivazione ODbL aggiungerebbe complessità senza un beneficio necessario al target Plausible+.

## B.6 Gap di evidenza effettivamente rimasti

**UNRESOLVED EVIDENCE GAP:** non ho verificato una serie nazionale aperta, acquisibile e confrontabile che consenta di costruire direttamente le distribuzioni dei first names per **1950–1998**.

**UNRESOLVED EVIDENCE GAP:** non ho verificato un dataset nazionale di prevalenza dei cognomi che soddisfi contemporaneamente copertura, acquisizione e condizioni di riuso richieste.

Queste affermazioni descrivono l’esito della verifica, **non dimostrano l’inesistenza assoluta di altre fonti**. Entrambi i gap hanno una soluzione curatoriale già scelta; non lasciano a Codex la scelta della strategia.

---

# C. D1-B First-Name Curation Contract

## C.1 Cardinalità e quote

Strategia complessiva: **EVIDENCE-INFORMED CURATED**.

| GenderCohortRigheWeight 4Weight 2Weight 1 |             |         |         |         |         |
| ----------------------------------------- | ----------- | ------- | ------- | ------- | ------- |
| `female`                                  | `1950-1964` | 80      | 20      | 40      | 20      |
| `female`                                  | `1965-1979` | 80      | 20      | 40      | 20      |
| `female`                                  | `1980-1989` | 80      | 20      | 40      | 20      |
| `female`                                  | `1990-1999` | 80      | 20      | 40      | 20      |
| `female`                                  | `2000-2005` | 80      | 20      | 40      | 20      |
| `male`                                    | `1950-1964` | 80      | 20      | 40      | 20      |
| `male`                                    | `1965-1979` | 80      | 20      | 40      | 20      |
| `male`                                    | `1980-1989` | 80      | 20      | 40      | 20      |
| `male`                                    | `1990-1999` | 80      | 20      | 40      | 20      |
| `male`                                    | `2000-2005` | 80      | 20      | 40      | 20      |
| **Totale**                                |             | **800** | **200** | **400** | **200** |

Le 800 righe **non devono rappresentare 800 nomi distinti**: la persistenza dello stesso nome fra coorti è ammessa e desiderata.

**Rationale.** Sessanta nomi soddisfano il minimo, ma lasciano meno spazio alla varietà. Ottanta aggiungono un terzo delle entry rispetto al minimo. Cento aumenterebbero ulteriormente del 25% il lavoro sulle righe senza essere necessari per l’unicità tecnica, già affidata ai token di Email e SiteURL. La scelta normativa è **80**, non un intervallo.

Per ciascun gruppo:

```math
W=20\cdot4+40\cdot2+20\cdot1=180
```

| TierMassa esattaPercentuale |       |           |
| --------------------------- | ----- | --------- |
| `common`, weight 4          | `4/9` | 44,4444…% |
| `regular`, weight 2         | `4/9` | 44,4444…% |
| `rare`, weight 1            | `1/9` | 11,1111…% |

La probabilità massima di una singola entry, condizionata al gruppo, è `4/180 = 1/45`, circa **2,22%**.

**Queste sono probabilità del generatore, non stime delle frequenze italiane.**

## C.2 Metodologia per coorte

| CohortEvidenza usataProcedura normativaRanking e pesi |                                                                          |                                                                                  |                                                               |
| ----------------------------------------------------- | ------------------------------------------------------------------------ | -------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `1950-1964`                                           | Riferimento onomastico storico e giudizio intergenerazionale documentato | Protocollo curatoriale C.3; nessuna retroproiezione ISTAT                        | Ranking curatoriale; posizioni 1–20 → 4, 21–60 → 2, 61–80 → 1 |
| `1965-1979`                                           | Stesso riferimento, valutando specificamente questa generazione          | C.3; non copiare la lista della coorte precedente                                | Stessa procedura, con giudizi riferiti alla coorte            |
| `1980-1989`                                           | Riferimento onomastico e valutazione delle transizioni generazionali     | C.3; i dati successivi non sostituiscono quelli mancanti                         | Stessa procedura                                              |
| `1990-1999`                                           | Curation del decennio; ISTAT 1999 come riscontro finale parziale         | C.3; i conteggi 1999 non vengono moltiplicati né usati come ranking del decennio | Ranking curatoriale, non ranking 1999                         |
| `2000-2005`                                           | Conteggi ISTAT dei sei anni, per sesso                                   | Protocollo empirico limitato C.4                                                 | Somma dei conteggi; stesse fasce ordinali 20/40/20            |

## C.3 Protocollo curatoriale per le prime quattro coorti

Si impiegano **due persone**, A e B, italofone e con familiarità documentata con il repertorio italiano intergenerazionale. A può essere il curatore; B deve effettuare una valutazione indipendente prima di vedere i punteggi di A.

Per ciascun `gender × cohort` si congela una pool di **120 candidati ammissibili**. Le proposte respinte restano nel ledger, ma non contano nei 120.

La pool nasce da proposta editoriale umana, consultazione dei riferimenti dichiarati e confronto intergenerazionale. Non nasce dall’estrazione delle liste Treccani, da directory personali o da un repertorio generato e approvato esclusivamente da un LLM.

Per ogni candidato, entrambi valutano separatamente:

| CampoCondizione di ammissibilità |                                                                                         |
| -------------------------------- | --------------------------------------------------------------------------------------- |
| `orthography_fit`                | La forma è riconosciuta come nome, con grafia plausibile                                |
| `gender_fit`                     | L’associazione al gruppo è sostenuta esplicitamente                                     |
| `era_fit`                        | Il nome è plausibile per una persona della coorte, non soltanto per una persona di oggi |
| `familiarity_score`              | Punteggio della tabella seguente                                                        |

| ScoreSignificato curatoriale |                                                                             |
| ---------------------------- | --------------------------------------------------------------------------- |
| 0                            | Non sostenibile, anacronistico o non riconosciuto con sufficiente sicurezza |
| 1                            | Plausibile, ma poco familiare per quella generazione                        |
| 2                            | Familiare per quella generazione                                            |
| 3                            | Fortemente familiare e candidato credibile al tier common                   |

Un candidato entra nella pool ammissibile soltanto con tutti i `fit=true` e score almeno 1 per entrambi.

L’ordinamento è:

1. `min(score_A, score_B)` decrescente;
2. `score_A + score_B` decrescente;
3. digest di spareggio curatoriale crescente;
4. chiave lessicale in ordine UTF-8 crescente.

Lo spareggio è congelato come:

```text
SHA-256(
  UTF-8(
    "it-2026a|D1-B|curation-tie-v1|{dataset}|{lexical-key}"
  )
)
```

I literal `dataset` sono `first-names`, `last-names`, `street-names`.

Per i first names, `lexical-key = gender + "|" + D(name)`, dove `D` è definita in G.2. **La coorte è deliberatamente esclusa dallo spareggio:** giudizi identici sullo stesso nome non devono produrre cambiamenti generazionali dovuti soltanto al digest.

Si selezionano le prime **80** entry e si assegnano le fasce 20/40/20. Tutte le 20 entry common devono inoltre avere:

```text
min(score_A, score_B) >= 2
max(score_A, score_B) = 3
```

Se la condizione non è soddisfatta, si rivedono candidati ed evidenze. **Non si abbassa la soglia e non si promuovono nomi soltanto per completare la quota.**

Ogni giudizio 3, ogni composto e ogni associazione a entrambi i gender richiede una nota originale di motivazione. Per gli altri casi sono sufficienti valutazioni individuali e note metodologiche a livello di gruppo.

La riproducibilità riguarda il **ledger congelato**: stessi candidati, giudizi ed evidenze producono lo stesso output. Non si pretende che due nuove persone ricostruiscano spontaneamente la stessa lista.

## C.4 Protocollo per `2000-2005`

Per ciascun gender si congela una pool di **120 candidati ammissibili**.

La pool deve includere l’unione delle prime **20 posizioni** delle graduatorie annuali ISTAT 2000–2005, limitatamente alle entry ammissibili. Se una graduatoria pubblicata contiene meno di 20 posizioni, si considerano tutte quelle presenti. Le restanti posizioni della pool vengono proposte dal curatore e verificate mediante la serie del nome.

Una voce di quella unione può essere esclusa soltanto per un problema documentato di grafia, associazione al sesso, ammissibilità del display o supporto ASCII. L’esclusione non può servire ad alterare la distribuzione desiderata.

Per ogni candidato si acquisiscono **sei conteggi annuali**, con ambito nazionale e senza selezione per cittadinanza.

```math
S(n,g)=\sum_{y=2000}^{2005} count(n,g,y)
```

Il ranking usa esclusivamente `S` decrescente, con spareggio tramite `D(name)` in ordine UTF-8. Non si usano media dei ranghi, massimo annuale o frequenze del 2024.

Sono richiesti sei valori numerici espliciti e `S>0`. **Assente, non restituito o soppresso non significa zero.** Lo zero è ammesso soltanto quando è effettivamente fornito come valore numerico dalla fonte.

La procedura di acquisizione è: esportazione ufficiale utilizzabile, quando disponibile; altrimenti trascrizione dei valori visualizzati, verificata integralmente da una seconda persona. Si conservano parametri della consultazione, etichetta originale, data e riscontro della trascrizione. Non si impone un’API non verificata.

Le varianti linguisticamente distinte non vengono sommate. La corrispondenza fra etichetta della fonte e display deve essere documentata; una corrispondenza ambigua rende il candidato non ammissibile.

Si selezionano le prime **80** entry e si assegnano:

```text
rank  1–20 → weight 4
rank 21–60 → weight 2
rank 61–80 → weight 1
```

**ACCEPTABLE LIMITATION:** questo è il ranking di una pool curata e verificata, non la dichiarazione di aver ricostruito gli 80 nomi nazionali più frequenti in assoluto.

## C.5 Gender, nomi moderni e composti

Il mapping è esattamente:

```text
female → GenderFemale
male   → GenderMale
```

Il campo identifica il bucket runtime. Non autorizza a inferire il gender dallo spelling né a interpretare una registrazione anagrafica isolata come prova sufficiente: il prodotto ISTAT segnala esplicitamente la possibile presenza di incompatibilità nome–sesso. ([Istat](https://www.istat.it/dati/calcolatori/contanomi/ "https://www.istat.it/dati/calcolatori/contanomi/"))

Un nome presente in entrambi i gruppi richiede approvazione esplicita di A e B e una nota `cross_gender_justification`; dove disponibili, si riportano separatamente i riscontri empirici per i due sessi.

Per le coorti moderne non esiste una whitelist “traditional Italian only”. Sono ammissibili nomi di diversa origine linguistica quando soddisfano gli stessi criteri di evidenza, coorte e grafia. **Non si introducono quote o campi per ethnicity, immigration status o religion.**

Gli spazi nei composti sono ammessi. `Gian Luca` e `Gianluca` sono entry distinte soltanto se entrambe sono sostenute come forme complete. Lo stesso vale per `Anna Maria`, `Maria Chiara` e `Pier Paolo`.

Non si generano composti combinando parti. Non si uniscono conteggi di forme unite e separate. Il normale algoritmo ASCII produce, per esempio:

```text
Gian Luca → gian-luca
Gianluca  → gianluca
```

## C.6 Differenziazione delle coorti

Per ogni gender si definiscono:

- `S_c`: insieme delle 80 chiavi `D(name)` della coorte;
- `C_c`: insieme delle 20 chiavi con weight 4.

Si applicano le seguenti soglie **curatoriali**, non statistiche:

| ControlloSoglia normativaMotivazione                                       |        |                                                           |
| -------------------------------------------------------------------------- | ------ | --------------------------------------------------------- |
| Overlap del vocabolario fra qualsiasi coppia di coorti dello stesso gender | \`     | S\_a ∩ S\_b                                               |
| Overlap common fra coorti adiacenti                                        | \`     | C\_a ∩ C\_b                                               |
| Overlap del vocabolario fra `1950-1964` e `2000-2005`                      | `≤ 56` | Almeno il 30% del vocabolario deve differire agli estremi |
| Overlap common fra le due coorti estreme                                   | `≤ 10` | Almeno metà del tier common deve cambiare                 |

Non esiste una soglia minima di overlap: la persistenza è consentita, non artificialmente imposta.

Il report deve mostrare tutte le **10 coppie per gender**, non soltanto quelle adiacenti. Gli ingressi/uscite dal common vengono revisionati integralmente.

Questi controlli non rilevano da soli gli anacronismi. È inoltre obbligatorio che ogni entry selezionata abbia entrambi gli `era_fit=true`, e che la review segnali come `ISSUE` una differenziazione ottenuta soltanto alternando grafie, spazi o varianti dello stesso repertorio.

Un fallimento richiede una **curation correction** motivata. È vietato eliminare o spostare automaticamente nomi per superare le soglie.

---

# D. D1-B Surname Curation Contract

## D.1 Strategia, dimensione e distribuzione

Strategia: **ORIGINAL CURATED**.

“Original” riguarda la selezione, l’organizzazione e le annotazioni del repertorio. **Non significa inventare cognomi nuovi.**

Il dataset contiene esattamente **600 cognomi**. Si sceglie il minimo architetturale: 750 aumenterebbero il lavoro del 25%, 1.000 di circa il 67%, senza che sia necessario risolvere l’omonimia attraverso la lista.

| WeightRighePeso totaleMassa |         |           |          |
| --------------------------- | ------- | --------- | -------- |
| 4                           | 120     | 480       | 40%      |
| 2                           | 240     | 480       | 40%      |
| 1                           | 240     | 240       | 20%      |
| **Totale**                  | **600** | **1.200** | **100%** |

La massima probabilità per entry è `4/1200 = 1/300`, circa **0,3333%**.

Il 40% della probabilità va ai cognomi più familiari, ma il 60% resta fuori dal common. Il tier rare non viene presentato come una coda statistica nazionale misurata.

## D.2 Procedura curatoriale

Si congela una pool di **750 candidati ammissibili**, proposta editorialmente prima del confronto con liste di frequenza.

Sono vietate invenzioni tramite prefissi, suffissi o combinazioni; ogni cognome deve essere riconosciuto da entrambi i valutatori come cognome esistente e plausibile nel repertorio italiano.

Si usa il protocollo a due valutatori di C.3, sostituendo la familiarità generazionale con:

```text
3 = fortemente familiare nel repertorio nazionale
2 = familiare nel repertorio italiano
1 = riconosciuto come esistente e plausibile, meno familiare
0 = non sufficientemente sostenuto
```

Il ranking usa `min`, somma dei punteggi, digest curatoriale e chiave lessicale. Qui `lexical-key = D(name)`.

Si selezionano le prime 600 entry:

```text
rank   1–120 → weight 4
rank 121–360 → weight 2
rank 361–600 → weight 1
```

Le entry common devono avere `min≥2` e `max=3`, come nel protocollo storico.

La voce Treccani *Cognomi* è un controllo di struttura e ortografia, non una lista da trasferire. Non si incorporano ranghi o percentuali locali come se fossero nazionali.

La review deve rilevare una lista dominata da un solo repertorio locale, ma **non si introducono campi geografici, quote regionali o correlazioni surname–place**.

## D.3 Ortografia

Si conservano particelle, spazi e apostrofi:

```text
De Luca
Di Stefano
D'Angelo
D'Amico
Lo Monaco
La Rosa
Del Vecchio
De Santis
```

Per queste forme, la capitalizzazione mostrata è canonica. Non si applica automaticamente `Title Case` a qualsiasi cognome: altre grafie vengono approvate esplicitamente nel ledger.

L’apostrofo nel display finale è sempre **U+0027**. Gli spazi sono singoli **U+0020**. Il display è NFC.

Non si trasformano `De Luca` in `Deluca` o `D'Angelo` in `Dangelo`. Una forma unita può essere un’altra entry soltanto se è sostenuta indipendentemente, non perché prodotta dalla normalizzazione.

Unicità, ordinamento, schema e verifica dell’atom sono quelli comuni della sezione G.

---

# E. D1-B Street Fragment Curation Contract

## E.1 Strategia e dimensione

Strategia: **ORIGINAL CURATED**.

Il dataset contiene esattamente **400 frammenti**. I sei tipi producono già **2.400 combinazioni tipo–frammento**, prima di considerare civici e suffissi.

Le quote sono:

```text
weight 4:  80
weight 2: 160
weight 1: 160
```

Peso totale: `800`. Masse: **40% / 40% / 20%**. Probabilità massima di un singolo frammento: `4/800 = 0,5%`.

Non si replica la frequenza reale di `Roma`, `Garibaldi` o `Dante`. Il criterio è familiarità toponomastica con diversità sintetica.

## E.2 Taxonomy e quote esatte

La taxonomy è esclusivamente **build-only**.

| Category literalContenutoPool ammissibileRighe finaliW4W2W1 |                                                                  |         |         |        |         |         |
| ----------------------------------------------------------- | ---------------------------------------------------------------- | ------- | ------- | ------ | ------- | ------- |
| `public_history`                                            | Figure storiche e civiche                                        | 72      | 60      | 12     | 24      | 24      |
| `arts_letters`                                              | Letteratura, arti e musica                                       | 96      | 80      | 16     | 32      | 32      |
| `science_technology`                                        | Scienza, ricerca e tecnica                                       | 48      | 40      | 8      | 16      | 16      |
| `geography`                                                 | Denominazioni geografiche                                        | 72      | 60      | 12     | 24      | 24      |
| `civic_values`                                              | Concetti nazionali e civici                                      | 48      | 40      | 8      | 16      | 16      |
| `dates_events`                                              | Date ed eventi storici                                           | 24      | 20      | 4      | 8       | 8       |
| `nature`                                                    | Flora, natura e paesaggio                                        | 72      | 60      | 12     | 24      | 24      |
| `traditional_local`                                         | Toponimia tradizionale, agionimi ed espressioni locali generiche | 48      | 40      | 8      | 16      | 16      |
| **Totale**                                                  |                                                                  | **480** | **400** | **80** | **160** | **160** |

Ogni candidato ha **una sola categoria**, approvata nel ledger. Nei casi ambigui, l’assegnazione è editoriale e viene congelata prima del ranking; il builder non la deduce dal testo.

I candidati vengono creati come espressioni complete, non estratti da un elenco di strade. La consultazione di ANNCSU serve al confronto, non a popolare automaticamente la pool.

Si applica il ranking curatoriale di C.3 **all’interno di ciascuna categoria**. Qui i punteggi esprimono familiarità e naturalezza toponomastica, non frequenza reale; `lexical-key = D(name)`.

Si selezionano le entry e si applicano le fasce della matrice. Per le entry common valgono `min≥2` e `max=3`.

## E.3 Compatibilità con i tipi

Ogni frammento selezionato deve funzionare ragionevolmente con **tutti e sei** i tipi:

```text
Via
Viale
Piazza
Corso
Largo
Vicolo
```

“Funzionare” significa formare una denominazione sintetica naturale, non dimostrare che quella strada esista.

Il report completo contiene **400 righe × 6 combinazioni**. Entrambi i valutatori devono approvarne la compatibilità.

Non si aggiungono liste di tipi consentiti al runtime, filtri, retry o eccezioni. Un frammento non approvato per uno dei sei tipi viene corretto o sostituito nella curation.

I pesi runtime dei tipi restano quelli congelati: **55 / 15 / 12 / 8 / 5 / 5**.

## E.4 Display canonico

Il frammento contiene soltanto la denominazione successiva al tipo.

| AspettoRegola           |                                                                                  |
| ----------------------- | -------------------------------------------------------------------------------- |
| Nomi personali          | Forma estesa: `Giuseppe Garibaldi`, non `G. Garibaldi`                           |
| Figure dedicate         | Figure storiche; non si costruisce un repertorio di personalità viventi          |
| Preposizioni e articoli | Conservati, normalmente minuscoli: `della Repubblica`, `dei Tigli`               |
| Nomi propri             | Capitalizzazione editoriale approvata; nessun title-casing automatico            |
| San/Santa               | Forme estese `San`, `Santa`, `Santo`, `Sant'`; nessuna abbreviazione con punto   |
| Apostrofo               | U+0027                                                                           |
| Trattino                | U+002D, soltanto se parte della forma approvata                                  |
| Numerali                | Lettere ASCII maiuscole, non caratteri Unicode “Roman numeral”                   |
| Date                    | Giorno in numeri romani e mese esteso: `XX Settembre`, `XXV Aprile`, `II Giugno` |
| Anni nelle date         | Esclusi in v1                                                                    |
| Cifre arabe             | Escluse dall’intero frammento                                                    |
| Indirizzo completo      | Vietati tipo, civico, CAP, Comune aggiunto come campo d’indirizzo e coordinate   |

Una denominazione geografica come `Roma` è consentita come toponimo; è vietato aggiungere una città per trasformare il frammento in un indirizzo completo.

Ogni frammento con un token numerale romano viene annotato nel ledger con `numeral_role = date` oppure `regnal`. Gli altri hanno `numeral_role = none`. L’uso del numerale come civico è vietato.

Il controllo del prefisso confronta il primo token, con `D`, con questo insieme esatto:

```text
via, viale, piazza, corso, largo, vicolo,
strada, piazzale, piazzetta, lungomare, contrada, località
```

La presenza di uno di questi token all’inizio rende la riga non valida. Il divieto di punti elimina anche le abbreviazioni usuali dei tipi.

---

# F. ASCII Atom v1

## F.1 Scopo, versione e assunzioni

`ascii_atom` è contenuto normativo precomputato di `first_names.csv` e `last_names.csv`. Non è una identity key. `street_names.csv` non contiene un atom.

Il runtime non esegue normalizzazione o translitterazione.

Versione normativa: **Unicode 16.0.0**, con NFD/NFC definiti da UAX #15, revisione 56, e proprietà del relativo Unicode Character Database. La scelta è un pin di versione, non una dichiarazione di usare sempre l’Unicode più recente. ([Unicode](https://www.unicode.org/reports/tr15/tr15-56.html "https://www.unicode.org/reports/tr15/tr15-56.html"))

Una libreria è utilizzabile soltanto se conforme a queste regole. Nome e versione effettivi dello strumento offline devono essere registrati.

## F.2 Trasformazione esatta

L’input è decodificato come UTF-8 rigoroso. Byte non validi producono `ERROR_INVALID_UTF8`.

Si applica **NFD**, non NFKD. Si percorre poi la sequenza normalizzata nell’ordine risultante.

| Input dopo NFDOperazione                                                           |                                                                                               |
| ---------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| ASCII `A`–`Z`                                                                      | Convertire in `a`–`z`                                                                         |
| ASCII `a`–`z`, `0`–`9`                                                             | Conservare                                                                                    |
| Mark Unicode in `U+0300–U+036F`, escluso `U+034F`, con categoria `Mn`, `Mc` o `Me` | Rimuovere soltanto se appartiene a una sequenza immediatamente successiva a una lettera ASCII |
| Apostrofi `U+0027`, `U+2018`, `U+2019`, `U+02BC`                                   | Rimuovere, senza inserire un separatore                                                       |
| Spazi dell’insieme seguente                                                        | Emettere `-`                                                                                  |
| Trattini dell’insieme seguente                                                     | Emettere `-`                                                                                  |
| Punto ASCII `U+002E`                                                               | Emettere `-`                                                                                  |
| Qualsiasi altro carattere                                                          | `ERROR_UNSUPPORTED`                                                                           |

Insieme esatto degli spazi:

```text
U+0020, U+00A0, U+1680,
U+2000–U+200A,
U+202F, U+205F, U+3000
```

Insieme esatto dei trattini:

```text
U+002D, U+2010, U+2011, U+2012, U+2013, U+2014
```

Un mark consentito senza una lettera ASCII base immediatamente precedente — eventualmente seguita da altri mark consentiti — produce `ERROR_ORPHAN_MARK`. Apostrofi, cifre e separatori interrompono tale relazione con la base.

Dopo la scansione si comprimono tutte le sequenze di `-` in un solo `-` e si eliminano i `-` iniziali e finali. Un risultato vuoto produce `ERROR_EMPTY`.

Non si esegue full Unicode case folding per generare l’atom. Non esistono tabelle implicite per `ß`, `Ł`, `Ø`, `Æ` o `Œ`: se rimangono dopo NFD, sono non supportati.

Controlli, tab, newline, caratteri zero-width, emoji, slash, underscore, virgole e altra punteggiatura non elencata causano errore. Non vengono silenziosamente rimossi.

In caso di errore non si emette un atom parziale, non si applica un fallback e non si modifica il display per aggirare il problema.

## F.3 Invarianti e collisioni

Il risultato deve soddisfare:

```text
^[a-z0-9]+(?:-[a-z0-9]+)*$
```

Per ogni riga:

```text
stored_ascii_atom == ASCII_Atom_v1(name)
```

La first initial è **il primo byte dell’atom del first name**, non il primo carattere Unicode del display e non una ricerca della prima lettera successiva.

La policy delle collisioni è:

```text
ALLOWED + REPORTED
```

Display distinti che producono lo stesso atom rimangono distinti, purché la loro grafia sia sostenuta. Non si aggiungono suffissi artificiali. L’audit elenca atom, display distinti e chiavi interessate.

La semplice ricorrenza dello stesso display in più coorti non è una nuova collisione ortografica.

**ACCEPTABLE LIMITATION:** alcuni nomi plausibili con lettere latine non decomponibili vengono esclusi da questo repertorio v1. Non vengono riscritti con una grafia artificiale.

## F.4 Golden vectors

Nella tabella, `\uXXXX` indica il code point, non i caratteri letterali della sequenza di escape.

| InputOutput esatto  |                     |
| ------------------- | ------------------- |
| `Mario`             | `mario`             |
| `Nicolò`            | `nicolo`            |
| `Niccolò`           | `niccolo`           |
| `D'Angelo`          | `dangelo`           |
| `De Luca`           | `de-luca`           |
| `Maria Chiara`      | `maria-chiara`      |
| `Pier Paolo`        | `pier-paolo`        |
| `Jean-Pierre`       | `jean-pierre`       |
| `Maria Chiara`      | `maria-chiara`      |
| `D’Angelo` — U+2019 | `dangelo`           |
| `ÀÈÉÌÒÓÙ`           | `aeeioou`           |
| `NICOLO\u0300`      | `nicolo`            |
| `De\u00A0Luca`      | `de-luca`           |
| `Jean\u2011Pierre`  | `jean-pierre`       |
| `G. D’Annunzio`     | `g-dannunzio`       |
| `—Anna--Maria—`     | `anna-maria`        |
| `Giovan Battista`   | `giovan-battista`   |
| `Giovanbattista`    | `giovanbattista`    |
| `Trevisàn`          | `trevisan`          |
| `Radamès`           | `radames`           |
| `Łukasz`            | `ERROR_UNSUPPORTED` |
| `Søren`             | `ERROR_UNSUPPORTED` |
| `Straße`            | `ERROR_UNSUPPORTED` |
| `Ægir`              | `ERROR_UNSUPPORTED` |
| `Œnone`             | `ERROR_UNSUPPORTED` |
| `Марко`             | `ERROR_UNSUPPORTED` |
| `\u0301Anna`        | `ERROR_ORPHAN_MARK` |
| `Anna\u200BMaria`   | `ERROR_UNSUPPORTED` |
| `A/B`               | `ERROR_UNSUPPORTED` |
| `. - '`             | `ERROR_EMPTY`       |

I quattro casi `Giovan Battista`, `Giovanbattista`, `Trevisàn` e `Radamès` riprendono problemi di grafia presenti nella fonte onomastica consultata. Sono vettori tecnici, non un’approvazione automatica delle entry per una specifica coorte. ([Treccani](https://www.treccani.it/enciclopedia/antroponimi_%28Enciclopedia-dell%27Italiano%29/ "https://www.treccani.it/enciclopedia/antroponimi_%28Enciclopedia-dell%27Italiano%29/"))

Il normalizzatore accetta alcuni input non canonici per rendere verificabile la trasformazione. **Ciò non rende validi quegli input come display nei CSV:** spazi multipli, punti e trattini tipografici sono comunque respinti dal contratto del display.

---

# G. D1-B Audit Contract

## G.1 Schemi runtime e display comune

Gli header sono esattamente:

**`first_names.csv`**

```csv
gender,cohort,name,ascii_atom,weight
```

**`last_names.csv`**

```csv
name,ascii_atom,weight
```

**`street_names.csv`**

```csv
name,weight
```

Non si aggiungono colonne build-only.

I literal delle coorti sono esclusivamente:

```text
1950-1964
1965-1979
1980-1989
1990-1999
2000-2005
```

I campi `weight` sono serializzati soltanto come `1`, `2` o `4`: niente `01`, `+1`, `4.0` o etichette testuali.

Tutti i display devono essere non vuoti, NFC, senza spazi iniziali/finali, con soli spazi U+0020 singoli, apostrofi U+0027 e trattini U+002D. Sono vietati controlli, newline, punti, cifre e punteggiatura diversa da spazio, apostrofo e trattino.

Le lettere e i mark del display devono essere compatibili con il repertorio di ASCII Atom v1. Per le strade questo controllo verifica il repertorio testuale, **senza introdurre una colonna atom**.

Le modifiche necessarie a rendere canonica una proposta sono operazioni di curation registrate. Il builder non “ripulisce” silenziosamente le righe approvate.

## G.2 Duplicate semantics e chiavi

Si definisce:

```text
D(name) = NFC(DefaultFullCaseFold_Unicode16(NFC(name)))
```

Il full case folding usa i mapping `C` e `F` di `CaseFolding.txt`, senza mapping Turkic `T`. **Questa funzione serve alle chiavi, non alla generazione degli atom.** Il file normativo è disponibile nell’UCD 16.0.0. ([Unicode](https://www.unicode.org/Public/16.0.0/ucd/ "https://www.unicode.org/Public/16.0.0/ucd/"))

| DatasetChiave di unicità e stable key per la review |              |
| --------------------------------------------------- | ------------ |
| First names                                         | \`gender + " |
| Last names                                          | `D(name)`    |
| Street fragments                                    | `D(name)`    |

La chiave conserva accenti, spazi, apostrofi e trattini. Non usa l’atom.

Quindi differenze solo di maiuscole o composizione Unicode non creano entry nuove; `Gian Luca` e `Gianluca` non vengono invece fuse.

Il carattere `|` non è ammesso nei display, perciò la composizione delle chiavi non è ambigua.

## G.3 Ordinamento e byte canonici

L’ordine dei first names è:

```text
gender: female, male
cohort: nell’ordine dei cinque literal sopra
UTF-8(D(name)) crescente
UTF-8(name) crescente come ultimo tie-break
```

Last names e street fragments usano:

```text
UTF-8(D(name)) crescente
UTF-8(name) crescente come ultimo tie-break
```

Non si usa un ordinamento dipendente dalla locale e non si ordina per peso.

I tre file devono avere UTF-8 senza BOM, solo LF, un LF finale, separatore virgola, header esatto, numero fisso di colonne e nessun record vuoto.

Il quoting è minimo e deterministico: un campo viene racchiuso fra doppi apici soltanto se contiene virgola, doppio apice, CR o LF; i doppi apici interni vengono raddoppiati. I controlli sul contenuto vietano comunque CR/LF nei campi e la punteggiatura non ammessa.

La riserializzazione canonica deve riprodurre **esattamente gli stessi byte**.

## G.4 Automatic validation

| AmbitoControlli obbligatori |                                                                                                                                                        |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Tutti i file                | Byte canonici, header, colonne, display valido, peso valido, unicità, ordine                                                                           |
| First names                 | Esattamente 10 gruppi; 80 righe per gruppo; quote 20/40/20; 800 righe totali; atom ricalcolato uguale; soglie di differenziazione rispettate           |
| First names storici         | Pool ammissibile di 120 per gruppo; valutazioni complete; ranking e fasce riproducibili; common conforme ai punteggi minimi                            |
| First names 2000–2005       | Pool di 120 per gender; sei conteggi espliciti per candidato; somma, ranking e selezione riproducibili; nessuna imputazione degli assenti              |
| Last names                  | 600 righe; quote 120/240/240; pool di 750; ranking riproducibile; atom ricalcolato uguale                                                              |
| Streets                     | 400 righe; quote 80/160/160; pool di 480; matrice categoria×peso esatta; nessun prefisso vietato; nessuna cifra araba; numerali coerenti con il ledger |
| Review                      | Valutazioni richieste presenti, firme legate agli hash correnti, nessuna issue aperta                                                                  |
| Provenance                  | Tutte le fonti e tutti gli input referenziati esistono; ruoli e condizioni di riuso sono registrati                                                    |

Le cardinalità si riferiscono alle righe dati, **header escluso**.

### Non-interferenza D1-A

Il builder D1-B deve trattare come read-only:

```text
age_weights.csv
places.csv
postal_codes.csv
```

Per ciascuno deve risultare:

```text
SHA-256(before)
  == SHA-256(after)
  == SHA-256(approved D1-A baseline)
```

L’hash atteso deve provenire dal manifest o dal commit D1-A approvato, non da una nuova baseline creata durante il build D1-B. Baseline mancante o confronto fallito impediscono il PASS. Il vincolo deriva esplicitamente dal perimetro frozen.

## G.5 Distribution audit

### First names

Per ciascuno dei dieci gruppi il report mostra cardinalità, conteggi per peso, peso totale, masse teoriche e **tutti i 20 nomi common**.

Seguono le matrici di overlap dei vocabolari e dei common, più una tabella per ogni nome con i cinque pesi:

```text
1950-1964 | 1965-1979 | 1980-1989 | 1990-1999 | 2000-2005
```

In questa sola tabella, `0` significa “entry assente dal dataset della coorte”; **non significa conteggio ISTAT zero**.

Per ogni passaggio adiacente:

```text
entering common   = nuovo peso 4, precedente peso diverso da 4
leaving common    = precedente peso 4, nuovo peso diverso da 4
persistent common = peso 4 in entrambe
```

Si mostrano inoltre tutti gli aumenti/diminuzioni di peso e i gruppi con maggiore overlap.

### Last names

Il report mostra 600 righe, quote 120/240/240, massa 40/40/20, tutti i common, collisioni degli atom e casi ortografici.

### Streets

Il report mostra 400 righe, quote 80/160/160, massa 40/40/20, matrice categoria×peso, massa per categoria e riscontri della compatibilità con i sei tipi.

**Non vengono inventati risultati osservati in questo documento:** liste common, overlap effettivi e collisioni saranno prodotti dalle future righe curate.

## G.6 Deterministic human-review sampling

Ogni campione non esaustivo del report usa esattamente la formula richiesta:

```text
SHA-256(
  UTF-8(
    "it-2026a|D1-B|review-v1|{category}|{stable-key}"
  )
)
```

Si ordina per **raw digest bytes ascending**, poi per stable key in ordine UTF-8 crescente. Si prendono le prime `n` entry. Non si usa RNG.

| Ambito del campione`category` literalDimensione         |                                       |                     |
| ------------------------------------------------------- | ------------------------------------- | ------------------- |
| First names, separatamente per ciascun gruppo, weight 2 | `first-names/w2`                      | **12**              |
| First names, separatamente per ciascun gruppo, weight 1 | `first-names/w1`                      | **8**               |
| Last names, weight 2                                    | `last-names/w2`                       | **60**              |
| Last names, weight 1                                    | `last-names/w1`                       | **60**              |
| Streets, separatamente per ciascuna categoria, weight 2 | `street-names/{taxonomy-category}/w2` | **5 per categoria** |
| Streets, separatamente per ciascuna categoria, weight 1 | `street-names/{taxonomy-category}/w1` | **5 per categoria** |

Per le strade, `{taxonomy-category}` assume esclusivamente gli otto literal di E.2: il contratto determina quindi esattamente i 16 literal di campionamento street.

La base della review rapida è:

| DatasetContenuto esaustivoCampioniBase totale |                       |                |                                              |
| --------------------------------------------- | --------------------- | -------------- | -------------------------------------------- |
| First names                                   | 20 common × 10 gruppi | 20 × 10 gruppi | **400 righe**                                |
| Last names                                    | 120 common            | 60 + 60        | **240 righe**                                |
| Streets                                       | 80 common             | 40 + 40        | **160 frammenti**, resi con tutti i sei tipi |

Si aggiungono **esaustivamente**, senza ricampionamento:

**First names:** tutte le entry con spazio, apostrofo, trattino o caratteri non ASCII; tutte le associazioni cross-gender; tutte le collisioni; tutte le correzioni rispetto alla revisione precedente.

**Last names:** tutte le entry con spazi, apostrofi, trattini o caratteri non ASCII; tutte le collisioni e correzioni. In questo modo sono inclusi tutti i casi con particelle separate.

**Streets:** tutti i frammenti con apostrofi, trattini, caratteri non ASCII o numerali; tutti quelli iniziali con `San`, `Santa`, `Santo` o `Sant'`; tutte le correzioni.

Le entry già mostrate non vengono duplicate: vengono referenziate per stable key. Il numero aggiuntivo degli edge case dipende deterministicamente dal contenuto, non da una scelta del reviewer.

Il report conserva inoltre l’appendice esaustiva dei **400 × 6** abbinamenti street. Il campione rapido non sostituisce l’approvazione della compatibilità dell’intero lexicon.

## G.7 PASS / ISSUE workflow

Gli esiti umani sono esclusivamente:

```text
PASS
ISSUE
```

Una review non ancora svolta resta priva di approvazione; non viene marcata PASS.

Ogni issue registra almeno:

```text
issue_id
dataset
stable_key oppure group_key
problem
source_or_curation_to_recheck
reviewer_id
```

La correzione viene registrata come **`curation correction`**, con valore precedente, valore nuovo e motivazione. Un conteggio ufficiale non viene alterato per far coincidere il dataset con il giudizio del reviewer.

Dopo una correzione si rigenerano validazione, ranking, distribuzioni e campioni. Vanno riesaminati tutti gli elementi modificati, i gruppi interessati e le nuove entry entrate nei campioni.

Il PASS finale richiede contemporaneamente:

```text
automatic validation: zero errori
human review: A e B approvano gli hash correnti
open issues: zero
D1-A hashes: invariati
```

Codex può proporre candidati e produrre i report. Non può inventare conteggi, attribuire giudizi umani non effettuati o firmare la review.

## G.8 Provenance e input congelati

Si congelano questi artifact:

| ArtifactContenuto obbligatorio        |                                                                                             |
| ------------------------------------- | ------------------------------------------------------------------------------------------- |
| `provenance/SOURCES-D1B.md`           | Source register, licenze, ruoli, attribuzioni e limitazioni                                 |
| `provenance/curation-contract.json`   | Versioni, cardinalità, quote, taxonomy, soglie, algoritmi e literal                         |
| `provenance/curation-ledger-d1b.json` | Candidati, esclusioni, display approvati, valutazioni, note, conteggi, ranking e correzioni |
| `provenance/evidence-d1b/`            | Snapshot e trascrizioni redistribuibili effettivamente usati, con hash                      |
| `provenance/build-audit-d1b.json`     | Hash degli input/output, controlli automatici, distribuzioni, collisioni e confronti D1-A   |
| `provenance/human-review-d1b.md`      | Report deterministico, campioni e appendici                                                 |
| `provenance/review-record-d1b.json`   | Reviewer, esiti, issue e approvazioni legate agli hash                                      |

Per ogni fonte si registrano nome, publisher, locator esatto, data di consultazione/acquisizione, licenza e versione, ruolo, modalità di acquisizione e limitazioni.

Il ruolo assume uno fra:

```text
empirical input
corroboration
inspiration/reference
```

Si registra inoltre se il contenuto è stato **copiato/estratto** oppure soltanto **consultato**.

Nel ledger, i conteggi ufficiali e i giudizi curatoriali rimangono campi distinti. Le note di curation non sono false annotazioni della fonte.

I rebuild usano gli input congelati, senza consultazioni HTTP live o sostituzioni automatiche con dati più recenti.

### Regola di distribuzione

I tre nuovi dataset vengono distribuiti sotto **CC BY 4.0**, con attribuzione alla curation e, per gli input empirici utilizzati, a ISTAT. Si dichiara espressamente che selezione, normalizzazione e pesi sono elaborazioni per dati sintetici, non un prodotto statistico ufficiale. Le condizioni pubblicate da ISTAT consentono il riuso con tali obblighi. ([Istat](https://www.istat.it/note-legali/ "https://www.istat.it/note-legali/"))

Le persone che contribuiscono alla curation devono autorizzare la distribuzione del proprio contributo. Non vengono incluse copie di fonti riservate, database commerciali o contenuti OSM.

Non si dichiara “sicura” una copia perché inferiore a una percentuale arbitraria. La coincidenza di parole comuni non sostituisce la documentazione del processo di selezione originale.

Questa regola riguarda i nuovi artifact D1-B; non cambia licenze o contenuti frozen di D1-A.

---

# H. Open risks e verifica del contratto

| RiskSeverityEvidenceRequired action                                                 |                           |                                                                                          |                                                                                            |
| ----------------------------------------------------------------------------------- | ------------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| Mancanza di conteggi nazionali verificati per le coorti storiche                    | **ACCEPTABLE LIMITATION** | ISTAT inizia nel 1999; le fonti storiche consultate non costituiscono la serie richiesta | Applicare C.3 e dichiarare sempre la natura curatoriale dei pesi                           |
| Anacronismi o differenze generazionali ottenute artificialmente                     | **REVIEW REQUIRED**       | Le soglie di overlap non dimostrano da sole realismo culturale                           | Approvare `era_fit`, common e transizioni; correggere la curation, non le soglie           |
| Acquisizione completa dei conteggi 2000–2005 non ancora eseguita                    | **REVIEW REQUIRED**       | Verificati prodotto e copertura; non acquisite in questa fase le serie delle pool finali | Congelare i sei valori per candidato e la verifica della trascrizione; nessuna imputazione |
| Assenza di una base nazionale pulita di prevalenza dei cognomi verificata           | **ACCEPTABLE LIMITATION** | La candidata comunale non soddisfa il requisito nazionale                                | Mantenere ORIGINAL CURATED; non presentare i pesi come prevalenza nazionale                |
| Copia accidentale di repertori riservati o import ODbL                              | **REVIEW REQUIRED**       | Fonti consultabili non equivalgono automaticamente a input redistribuibili               | Verificare source register, ledger e dichiarazioni dei curatori prima del rilascio         |
| Frammenti innaturali con uno dei tipi                                               | **REVIEW REQUIRED**       | La grammatica runtime non ha filtri di compatibilità                                     | Approvare tutti i 2.400 abbinamenti; sostituire i frammenti problematici                   |
| Esclusione di lettere non supportate da ASCII Atom v1                               | **ACCEPTABLE LIMITATION** | Decisione esplicita di non usare translitterazione universale                            | Segnalare le esclusioni; non alterare il display per aggirarle                             |
| Conformità dell’implementazione Unicode e non-interferenza D1-A ancora da attestare | **REVIEW REQUIRED**       | Builder, CSV finali e baseline hash non fanno parte di questa verifica                   | Eseguire goldens, controlli UCD 16.0.0 e confronti con gli hash approvati                  |

**Verifica svolta sul contratto:** cardinalità, somme delle quote, matrice street, masse probabilistiche e dimensioni dei campioni sono coerenti. I 30 golden sono stati riscontrati anche con un controllo locale sui caratteri utilizzati; l’ambiente locale usa UCD 15.1.0, quindi questo riscontro **non viene presentato come certificazione completa di conformità Unicode 16.0.0**.

Non sono stati prodotti i CSV completi né attestati risultati di human review o hash di file non disponibili. Non rimane una scelta curatoriale di strategia, quota, soglia, schema, algoritmo o campionamento da delegare a Codex.

```text
FIRST NAMES: READY
LAST NAMES: READY
STREET FRAGMENTS: READY
ASCII ATOM: READY

D1-B IMPLEMENTATION:
L1-D1-B READY FOR CURATION IMPLEMENTATION
```