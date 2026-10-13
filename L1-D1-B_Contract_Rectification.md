## 1. Rectification verdict

```text
R1 overlap thresholds: FIXED
R2 keys: FIXED
R3 human effort: FIXED
R4 modern-name acquisition: FIXED
R5 licensing boundary: FIXED
R6 ASCII Atom cleanup: FIXED
R7 execution workflow: FIXED
```

## 2. Normative replacement sections

Le disposizioni seguenti sostituiscono le sezioni o sottosezioni indicate. Restano invariati strategie, cardinalità finali, quote dei pesi, schemi runtime, serializzazione canonica e tutte le prescrizioni non esplicitamente sostituite. La sezione B sulle fonti non viene riaperta.

### C.2 Metodologia per coorte

Per `1950-1964`, `1965-1979` e `1980-1989` si applica il protocollo curatoriale C.3, usando i riferimenti storici già dichiarati. Non si retroproiettano dati moderni.

Per `1990-1999` si applica lo stesso protocollo. Il riscontro ISTAT del 1999 è corroborazione parziale: non determina il ranking del decennio e non viene esteso agli anni precedenti.

Per `2000-2005` si applica esclusivamente il protocollo `annual_top10_anchor_v1` di C.4. Le graduatorie annuali costituiscono un ancoraggio empirico; selezione finale e pesi rimangono parte di un repertorio **EVIDENCE-INFORMED CURATED**, non una ricostruzione statistica del top 80 nazionale.

Ogni gruppo conserva esattamente:

```text
80 selected rows
20 rows with weight 4
40 rows with weight 2
20 rows with weight 1
```

### C.3 Protocollo curatoriale per le prime quattro coorti

**Ruoli.** A è il curatore principale umano; B è un secondo reviewer umano, distinto da A. Entrambi devono essere italofoni e avere familiarità con il repertorio italiano intergenerazionale.

A svolge la curation dettagliata. B effettua controllo indipendente sui dati selezionati e sui sottoinsiemi ad alto rischio. **B non deve attribuire un secondo punteggio a tutti i candidati.**

**Pool.** Per ciascun `gender × cohort` si congela una pool di **120 candidati ammissibili**. Le proposte respinte restano nel ledger, ma non contano nei 120.

La pool deriva da proposta curatoriale e dai riferimenti già dichiarati. Proposte automatiche possono essere usate come bozze, ma non costituiscono evidenza né approvazione.

Per ogni candidato A registra:

```text
orthography_fit: true / false
gender_fit: true / false
era_fit: true / false
score_A: 0 / 1 / 2 / 3
```

Significato di `score_A`:

```text
0 = non sufficientemente sostenuto o non plausibile per la coorte
1 = plausibile, ma poco familiare per quella generazione
2 = familiare per quella generazione
3 = fortemente familiare; candidato credibile al common tier
```

Un candidato è ammissibile soltanto con tutti i fit uguali a `true` e `score_A >= 1`.

Le verifiche ortografiche e di associazione al gender possono essere riutilizzate per riferimento fra coorti quando riguardano la stessa forma. `era_fit` e `score_A` devono invece essere espliciti per ciascun gruppo: non vengono copiati automaticamente fra generazioni.

Ogni punteggio 3, composto, associazione cross-gender o caso che richieda una motivazione specifica deve avere una nota originale di A. Nessuna nota curatoriale viene presentata come dato della fonte.

**Ranking.** All’interno del gruppo, ordinare per:

```text
1. score_A descending
2. curation-tie digest ascending, compared as raw bytes
3. lexical-key ascending, compared as UTF-8 bytes
```

Il digest è:

```text
SHA-256(
  UTF-8(
    "it-2026a|D1-B|curation-tie-v1|{dataset}|{lexical-key}"
  )
)
```

I literal `dataset` sono:

```text
first-names
last-names
street-names
```

Per i first names, `lexical-key` è quella definita in G.2 e **non contiene la coorte**.

Si selezionano le prime 80 entry:

```text
rank  1–20 → weight 4
rank 21–60 → weight 2
rank 61–80 → weight 1
```

Tutte le 20 entry common devono avere `score_A = 3`. Se la pool non contiene almeno 20 candidati con tale punteggio, A deve rivedere candidati ed evidenze. Non si abbassa la soglia e non si promuovono candidati soltanto per completare la quota.

**Review indipendente.** B esamina tutte le **80 entry selezionate** del gruppo per plausibilità generazionale, associazione al gender, grafia e composizione complessiva. All’interno della stessa review approfondisce tutti i common, i campioni deterministici e gli edge case definiti in G.6.

B restituisce `PASS / ISSUE`, senza produrre `score_B`. Il controllo dell’intera lista e quello dei suoi sottoinsiemi possono avvenire nella stessa lettura: non sono richieste due revisioni separate della stessa entry.

La riproducibilità riguarda il ledger congelato: stessi candidati, valutazioni di A e riscontri producono la stessa selezione. Non si pretende che curatori diversi costruiscano spontaneamente lo stesso repertorio.

### C.4 Protocollo per `2000-2005`: `annual_top10_anchor_v1`

Per `it-2026a` è congelato **un solo metodo**, `annual_top10_anchor_v1`.

Un export ufficiale verificato può automatizzare l’acquisizione delle graduatorie. Non modifica il metodo di selezione e non introduce un diverso ramo basato sui conteggi. **Non è richiesta alcuna trascrizione manuale di frequenze annuali.**

#### Acquisizione dell’ancoraggio empirico

Per ciascun gender e ciascun anno dal 2000 al 2005 si acquisiscono le **prime 10 righe della graduatoria annuale ufficiale pubblicata**, nell’ordine mostrato dalla fonte.

Si conservano:

```text
year
gender
position_in_acquired_ranking
source_label
canonical_name oppure exclusion_reason
evidence_ref
```

`position_in_acquired_ranking` è l’indice della riga acquisita, da 1 a 10; non è un conteggio di nascite. Eventuali pari merito mantengono l’ordine della graduatoria congelata, senza estendere l’acquisizione oltre dieci righe.

Se una graduatoria pubblicata contiene effettivamente meno di dieci righe, si acquisiscono tutte quelle presenti. Una graduatoria non acquisita o non disponibile **non equivale a una graduatoria vuota**: genera `ISSUE`.

L’acquisizione usa una rappresentazione ufficiale pubblicamente accessibile: export verificato oppure lettura della graduatoria pubblicata. Non dipende da API private, endpoint non documentati o ricostruzioni tramite interrogazioni massive.

Per ogni vista si conservano locator, parametri, data, rappresentazione congelata consentita dalle condizioni di riuso e relativo hash.

Il massimo è:

```text
2 genders × 6 years × 10 observations = 120 name-position observations
```

B confronta le osservazioni acquisite e le eventuali esclusioni con le dodici viste ufficiali. Non verifica 1.440 conteggi.

#### Pool e valutazione curatoriale

Per ciascun gender A congela **120 candidati ammissibili**, usando fit e punteggi definiti in C.3.

La pool deve includere tutti i nomi ammissibili presenti nelle osservazioni acquisite. Le restanti posizioni sono proposte e valutate da A.

L’esclusione di un nome osservato richiede una motivazione relativa ad ammissibilità, grafia, associazione al gruppo o supporto ASCII. Non può servire soltanto a ottenere una distribuzione desiderata. B esamina tali esclusioni durante il controllo delle osservazioni.

Non si fondono grafie linguisticamente distinte. Una corrispondenza ambigua fra etichetta ufficiale e display canonico genera `ISSUE`.

Per ciascun candidato il tooling calcola:

```text
p(name, gender) =
    number of distinct years in 2000–2005
    in which the candidate occurs in the acquired annual top-10 observations
```

Quindi:

```text
0 <= p <= 6
```

`p = 0` significa soltanto “non osservato nelle posizioni acquisite”. **Non significa frequenza zero, assenza nella popolazione o scarsa plausibilità.**

#### Selezione deterministica

Un candidato è eleggibile per il common tier quando:

```text
score_A = 3

OR

score_A = 2 AND p >= 2
```

Devono esistere almeno 20 candidati eleggibili. In caso contrario si rivede la pool; non si inventano osservazioni e non si abbassa la soglia.

Ordinare i candidati eleggibili per:

```text
1. p descending
2. score_A descending
3. curation-tie digest ascending, compared as raw bytes
4. lexical-key ascending, compared as UTF-8 bytes
```

Selezionare i primi **20** e assegnare `weight = 4`.

Rimuovere questi 20 candidati dalla pool. Ordinare tutti i rimanenti per:

```text
1. score_A descending
2. p descending
3. curation-tie digest ascending, compared as raw bytes
4. lexical-key ascending, compared as UTF-8 bytes
```

Selezionare:

```text
first 40 remaining candidates → weight 2
next 20 remaining candidates  → weight 1
```

Digest e lexical-key sono quelli di C.3 e G.2.

Il risultato è esattamente **80 entry per gender**, con quote **20 / 40 / 20**. La persistenza nelle posizioni annuali più alte sostiene il common tier; le altre entry restano soggette a curation e review, senza essere presentate come il seguito di una graduatoria nazionale completa.

B esamina tutte le 80 entry selezionate per gender, applicando gli stessi approfondimenti di G.6. La presenza in una graduatoria non sostituisce la verifica di grafia, gender e plausibilità.

### C.6 Differenziazione delle coorti

Per ciascun gender:

```text
S(c) = set of D(name) keys among the 80 selected entries of cohort c
C(c) = set of D(name) keys among the 20 weight-4 entries of cohort c
```

Le quattro regole normative sono:

```text
1. ANY TWO DISTINCT COHORTS a, b OF THE SAME GENDER:
   |S(a) ∩ S(b)| <= 76

2. ADJACENT COHORTS a, b OF THE SAME GENDER:
   |C(a) ∩ C(b)| <= 18

3. EXTREME COHORTS OF THE SAME GENDER:
   |S(1950-1964) ∩ S(2000-2005)| <= 56

4. EXTREME COMMON TIERS OF THE SAME GENDER:
   |C(1950-1964) ∩ C(2000-2005)| <= 10
```

Le coorti adiacenti seguono l’ordine cronologico dei cinque literal del contratto.

Il limite 76 richiede almeno quattro differenze fra due vocabolari, senza imporre un ricambio esteso. Il limite 18 richiede almeno due ingressi e due uscite dal common fra coorti adiacenti. Agli estremi restano richieste differenze più visibili: almeno 24 entry nel vocabolario e dieci nel common.

Non esiste una soglia minima di overlap: la persistenza è ammessa, ma non imposta.

Il report mostra tutte le **dieci coppie per gender**, oltre a ingressi, uscite e persistenze del common. B controlla questi confronti insieme alle liste selezionate.

Ogni entry deve avere `era_fit = true` attestato da A e superare la review di B prevista per la lista selezionata. Le soglie non sostituiscono questo giudizio.

Un fallimento richiede una `curation correction` motivata. Non si eliminano o spostano automaticamente nomi per superare le soglie. Differenze ottenute soltanto alternando grafie o spaziature, senza una motivazione generazionale, costituiscono `ISSUE`.

### D.2 Procedura curatoriale dei cognomi

Si congela una pool di **750 candidati ammissibili**, proposta editorialmente. La strategia resta **ORIGINAL CURATED**: originalità della selezione e delle annotazioni, non invenzione di nuovi cognomi.

A verifica esistenza plausibile nel repertorio italiano, ortografia e ammissibilità di ogni candidato. Sono vietate invenzioni tramite combinazione di prefissi, suffissi o parti di cognomi.

A assegna:

```text
0 = non sufficientemente sostenuto
1 = riconosciuto come esistente e plausibile, meno familiare
2 = familiare nel repertorio italiano
3 = fortemente familiare nel repertorio nazionale
```

I 750 candidati ammissibili devono avere `score_A >= 1` e verifiche di ammissibilità positive.

Il ranking usa:

```text
1. score_A descending
2. curation-tie digest ascending, compared as raw bytes
3. lexical-key ascending, compared as UTF-8 bytes
```

Il literal dataset è `last-names`; `lexical-key = D(name)`.

Si selezionano:

```text
rank   1–120 → weight 4
rank 121–360 → weight 2
rank 361–600 → weight 1
```

Tutti i 120 common devono avere `score_A = 3`. Una carenza richiede revisione della pool, non abbassamento della soglia.

B non attribuisce punteggi ai 750 candidati. Esamina il repertorio attraverso tutti i common, i campioni regular/rare e tutti gli edge case selezionati definiti in G.6, restituendo `PASS / ISSUE`.

La review deve rilevare grafie implausibili, omissioni evidenti nel common e una composizione palesemente sbilanciata verso un solo repertorio locale. Non introduce quote regionali o correlazioni surname–place.

I riferimenti onomastici già dichiarati restano controlli di plausibilità e ortografia, non liste da trasferire. I punteggi non vengono presentati come prevalenze nazionali misurate.

### E.2 Taxonomy, pool e ranking dei frammenti

La taxonomy resta esclusivamente **build-only**.

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

Ogni candidato ha una sola categoria, assegnata da A prima del ranking. Il builder non la deduce dal testo.

A propone espressioni complete e verifica ammissibilità, ortografia e naturalezza toponomastica. Non si popola la pool tramite estrazione di un elenco di strade.

A assegna:

```text
0 = non sufficientemente naturale o sostenibile
1 = plausibile, ma meno familiare
2 = familiare e naturale come denominazione toponomastica
3 = fortemente familiare e naturale; candidato credibile al common tier
```

I 480 candidati ammissibili devono avere `score_A >= 1`.

All’interno di ciascuna categoria si ordina per:

```text
1. score_A descending
2. curation-tie digest ascending, compared as raw bytes
3. lexical-key ascending, compared as UTF-8 bytes
```

Il literal dataset è `street-names`; `lexical-key = D(name)`.

Si seleziona il numero finale della categoria e si assegnano, nell’ordine, le sue quote W4, W2 e W1. Tutti i common devono avere `score_A = 3`.

B non attribuisce un secondo punteggio ai 480 candidati. La sua review riguarda il sottoinsieme finale definito in G.6 e la composizione complessiva mostrata dal report.

### E.3 Compatibilità con i tipi

Ogni frammento selezionato deve formare una denominazione sintetica ragionevolmente naturale con tutti e sei i tipi:

```text
Via
Viale
Piazza
Corso
Largo
Vicolo
```

Non è richiesto dimostrare che le combinazioni esistano realmente.

**A controlla tutti i 400 × 6 abbinamenti.** Il report li presenta come 400 pannelli, ciascuno con le sei combinazioni. A può registrare un solo `PASS` per pannello, purché abbia controllato tutte e sei le combinazioni; un problema identifica il tipo interessato.

**B controlla tutte e sei le combinazioni soltanto per i frammenti nel proprio insieme obbligatorio di G.6.** Non deve duplicare automaticamente l’intera verifica dei 400 pannelli.

Un frammento con una combinazione non approvata deve essere corretto o sostituito. Non si introducono filtri, retry, eccezioni o matrici di compatibilità nel runtime.

I pesi dei tipi restano **55 / 15 / 12 / 8 / 5 / 5**. L’appendice completa dei 2.400 abbinamenti rimane disponibile anche quando B ne esamina soltanto il sottoinsieme prescritto.

### F.2.1 Stato dei mark e precedenza degli errori

*Nuova sottosezione di precisazione; non sostituisce l’algoritmo e gli insiemi di caratteri di F.2.*

La decodifica UTF-8 rigorosa precede la normalizzazione. Una sequenza di byte invalida produce `ERROR_INVALID_UTF8`, senza scansione parziale.

Dopo NFD, lo stato “base ASCII alfabetica disponibile” parte da `false`:

```text
ASCII letter:
    emit the prescribed lowercase letter
    set base state to true

permitted combining mark:
    require base state = true
    remove the mark
    preserve base state

digit, apostrophe or separator:
    apply the existing F.2 rule
    set base state to false
```

Un mark consentito con stato `false` produce `ERROR_ORPHAN_MARK`. U+034F resta non supportato e produce `ERROR_UNSUPPORTED`, non `ERROR_ORPHAN_MARK`.

La scansione si arresta al primo errore incontrato nella sequenza NFD. `ERROR_EMPTY` si valuta soltanto dopo una scansione riuscita, il collasso dei separatori e il trimming finale.

Non cambiano NFD, conversione delle sole maiuscole ASCII, insiemi di caratteri, rifiuto delle lettere non supportate o assenza di fallback.

### F.4 Golden vectors

La suite normativa contiene **37 vettori con input distinti**:

```text
22 successful transformations
10 ERROR_UNSUPPORTED
 3 ERROR_ORPHAN_MARK
 1 ERROR_EMPTY
 1 ERROR_INVALID_UTF8
```

`\uXXXX` indica il code point, non una sequenza letterale di backslash e cifre. Le virgolette delimitano l’input e non ne fanno parte. `bytes_hex` identifica byte grezzi.

Il caso canonico `Maria Chiara` compare una sola volta; il test degli spazi multipli usa un input distinto.

```text
01. "Mario"                 → "mario"
02. "Nicolò"                → "nicolo"
03. "Niccolò"               → "niccolo"
04. "D'Angelo"              → "dangelo"
05. "De Luca"               → "de-luca"
06. "Maria Chiara"          → "maria-chiara"
07. "Pier Paolo"            → "pier-paolo"
08. "Jean-Pierre"           → "jean-pierre"
09. "  Anna   Maria  "      → "anna-maria"
10. "D\u2019Angelo"         → "dangelo"
11. "ÀÈÉÌÒÓÙ"               → "aeeioou"
12. "NICOLO\u0300"          → "nicolo"
13. "De\u00A0Luca"          → "de-luca"
14. "Jean\u2011Pierre"      → "jean-pierre"
15. "G. D\u2019Annunzio"    → "g-dannunzio"
16. "\u2014Anna--Maria\u2014" → "anna-maria"
17. "Giovan Battista"       → "giovan-battista"
18. "Giovanbattista"        → "giovanbattista"
19. "Trevisàn"              → "trevisan"
20. "Radamès"               → "radames"

21. "Łukasz"               → ERROR_UNSUPPORTED
22. "Søren"                → ERROR_UNSUPPORTED
23. "Straße"               → ERROR_UNSUPPORTED
24. "Ægir"                 → ERROR_UNSUPPORTED
25. "Œnone"                → ERROR_UNSUPPORTED
26. "Марко"                → ERROR_UNSUPPORTED
27. "\u0301Anna"           → ERROR_ORPHAN_MARK
28. "Anna\u200BMaria"      → ERROR_UNSUPPORTED
29. "A/B"                  → ERROR_UNSUPPORTED
30. " . - ' "              → ERROR_EMPTY

31. "Nicolo"               → "nicolo"
32. "A\u0301\u0323"        → "a"
33. "1\u0301"              → ERROR_ORPHAN_MARK
34. "A'\u0301"             → ERROR_ORPHAN_MARK
35. "A\u034F"              → ERROR_UNSUPPORTED
36. "\uFB01"               → ERROR_UNSUPPORTED
37. bytes_hex("C3 28")     → ERROR_INVALID_UTF8
```

I vettori includono decomposizione canonica, riordinamento dei mark, interruzione della relazione con la base, collasso dei separatori e rifiuto di una ligatura che richiederebbe decomposizione di compatibilità.

La validità come input di test non implica validità come display nei CSV: rimangono applicabili le restrizioni di G.1.

Restano invariati:

```text
atom invariant:
    ^[a-z0-9]+(?:-[a-z0-9]+)*$

collision policy:
    ALLOWED + REPORTED

first initial:
    first byte of the successfully generated first-name ascii_atom
```

`Nicolò` e `Nicolo` dimostrano una collisione ammessa fra display distinti. Non si aggiungono suffissi. La first initial non viene ricavata dal display Unicode e non viene definita per trasformazioni fallite.

### G.2 Duplicate semantics, uniqueness keys e stable keys

Si conserva:

```text
D(name) = NFC(DefaultFullCaseFold_Unicode16(NFC(name)))
```

Il case folding usa i mapping `C` e `F`, senza mapping Turkic `T`. `D` serve alle chiavi, non alla generazione degli atom.

**First-name uniqueness key:**

```text
gender + "|" + cohort + "|" + D(name)
```

**First-name human-review stable key:**

```text
gender + "|" + cohort + "|" + D(name)
```

**First-name curation-tie lexical-key:**

```text
gender + "|" + D(name)
```

**Last-name uniqueness key e human-review stable key:**

```text
D(name)
```

**Street uniqueness key e human-review stable key:**

```text
D(name)
```

Per last names e street fragments, anche la curation-tie lexical-key è `D(name)`.

La coorte è quindi **obbligatoria** nelle chiavi di unicità e review dei first names, ma **deliberatamente assente** dallo spareggio curatoriale.

Sono ammesse entry con stesso display e stesso gender in coorti diverse, anche con pesi diversi. Non sono ammessi duplicati della stessa chiave all’interno di un gruppo.

La chiave conserva accenti, spazi, apostrofi e trattini; non usa `ascii_atom`. Il carattere `|` rimane vietato nei display, quindi i componenti non richiedono escaping.

Il campionamento umano usa esclusivamente le **human-review stable key**, mai le curation-tie lexical-key. Gruppi diversi restano distinguibili anche quando condividono il medesimo literal `category` di campionamento.

### G.4 Automatic validation

La validazione deve verificare:

| AmbitoControlli obbligatori |                                                                                                                                                                                                         |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Tutti i file                | Byte canonici, header e colonne esatti, display valido, peso valido, unicità secondo G.2 e ordinamento secondo G.3                                                                                      |
| First names                 | Dieci gruppi, 80 righe per gruppo, quote 20/40/20, 800 righe totali, atom ricalcolato uguale al memorizzato                                                                                             |
| First names storici         | Pool di 120 per gruppo, fit e `score_A` completi, ranking riproducibile, common con `score_A = 3`, soglie C.6 rispettate                                                                                |
| First names moderni         | Metodo `annual_top10_anchor_v1`, copertura delle dodici viste, massimo dieci osservazioni per vista, mapping/esclusioni espliciti, pool di 120 per gender, calcolo di `p` e selezione C.4 riproducibili |
| Last names                  | 600 righe, quote 120/240/240, pool di 750, ranking riproducibile, common con `score_A = 3`, atom corretto                                                                                               |
| Streets                     | 400 righe, quote 80/160/160, pool di 480, matrice categoria×peso esatta, ranking riproducibile, common con `score_A = 3`, controlli di display e numerali                                               |
| Review di A                 | Valutazioni richieste presenti; per le strade, approvazione di tutti i 400 pannelli da sei combinazioni                                                                                                 |
| Review di B                 | Copertura esatta dell’insieme obbligatorio corrente di G.6, controllo degli anchor moderni, nessuna issue aperta                                                                                        |
| Provenance                  | Input referenziati disponibili, ruoli e obblighi di riuso registrati, autorizzazioni dei contributori conservate                                                                                        |
| Approvazione finale         | Attestazioni umane riferite agli hash correnti; nessun PASS dedotto automaticamente dall’assenza di issue                                                                                               |

Non si richiede `score_B` e non si richiedono sei conteggi annuali per candidato moderno.

Il tooling verifica presenza, copertura e coerenza delle attestazioni umane; non certifica automaticamente la correttezza culturale del loro contenuto.

Le cardinalità escludono gli header.

**Non-interferenza D1-A.** Per ciascuno di `age_weights.csv`, `places.csv` e `postal_codes.csv` deve risultare:

```text
SHA-256(before)
    == SHA-256(after)
    == SHA-256(approved D1-A baseline)
```

La baseline deve provenire dal manifest o commit D1-A approvato, non essere creata durante D1-B. Baseline mancante o confronto fallito impediscono il PASS. Il vincolo di non-interferenza resta quello già congelato.

### G.6 Deterministic human-review sampling e copertura

Ogni campione non esaustivo usa:

```text
SHA-256(
  UTF-8(
    "it-2026a|D1-B|review-v1|{category}|{stable-key}"
  )
)
```

Si ordinano le entry per digest grezzo crescente, poi per stable key in ordine UTF-8 crescente. Si prendono le prime `n`. Non si usa RNG.

`stable-key` è sempre quella di **human review** definita in G.2. Le sostituzioni nella stringa non aggiungono spazi, newline o escaping.

#### Dimensioni dei campioni

| Dataset e ambitoCampione weight 2Campione weight 1     |    |    |
| ------------------------------------------------------ | -- | -- |
| First names, separatamente per ciascun gender × cohort | 12 | 8  |
| Last names                                             | 60 | 60 |
| Streets, separatamente per ciascuna categoria          | 3  | 3  |

I literal di campionamento sono esattamente:

```text
first-names/w2
first-names/w1

last-names/w2
last-names/w1

street-names/public_history/w2
street-names/public_history/w1
street-names/arts_letters/w2
street-names/arts_letters/w1
street-names/science_technology/w2
street-names/science_technology/w1
street-names/geography/w2
street-names/geography/w1
street-names/civic_values/w2
street-names/civic_values/w1
street-names/dates_events/w2
street-names/dates_events/w1
street-names/nature/w2
street-names/nature/w1
street-names/traditional_local/w2
street-names/traditional_local/w1
```

#### First names: copertura di B

B esamina **tutte le 80 entry selezionate per ciascuno dei dieci gruppi**.

All’interno della stessa review approfondisce:

```text
all 20 common entries per group
12 sampled regular entries per group
8 sampled rare entries per group
all compounds
all cross-gender cases
all entries containing apostrophes, hyphens or non-ASCII characters
all ascii_atom collision cases
all entries flagged review_attention = true by A
all relevant curation corrections
```

I sottoinsiemi si uniscono per stable key: una entry presente in più sottoinsiemi non richiede revisioni duplicate.

B controlla inoltre le matrici di overlap e le transizioni del common. Non attribuisce punteggi ai candidati esclusi dalla selezione.

#### Last names: copertura di B

L’insieme obbligatorio è l’unione di:

```text
all 120 common entries
60 sampled regular entries
60 sampled rare entries
all selected entries containing spaces, apostrophes, hyphens or non-ASCII characters
all other selected particle cases explicitly identified by A
all ascii_atom collision cases
all selected entries flagged review_attention = true by A
all relevant curation corrections
```

Spazi e annotazioni delle particelle devono assicurare l’inclusione dei casi ortografici previsti da D.3.

La base è **240 entry distinte**, cui si aggiungono soltanto gli edge case non già inclusi. Non è richiesta una seconda valutazione numerica dei 750 candidati.

#### Streets: copertura di B

L’insieme obbligatorio è l’unione di:

```text
all 80 common fragments
3 sampled regular fragments per category
3 sampled rare fragments per category
all 40 traditional_local fragments
all 20 dates_events fragments
all other fragments containing apostrophes, hyphens or non-ASCII characters
all other fragments with numeral_role != none
all other fragments beginning with San, Santa, Santo or Sant'
all selected fragments flagged review_attention = true by A
all relevant curation corrections
```

Per ogni frammento incluso B controlla **tutti e sei i tipi**, in un solo pannello.

Common e campioni producono 128 frammenti. L’inclusione esaustiva di `traditional_local` e `dates_events` porta la base a **164 frammenti distinti**, prima degli ulteriori edge case.

#### Registrazione senza lavoro duplicato

`review_attention` è un flag build-only impostato da A per casi che richiedano particolare attenzione culturale o ortografica. Non sostituisce i predicati obbligatori sopra elencati.

Sono consentite attestazioni `PASS` per un insieme esplicito di stable key, senza un clic separato per ogni entry, purché il reviewer abbia effettivamente esaminato tutti gli elementi attestati.

Common, campioni, edge case e correzioni vengono presentati come unione deduplicata. Il report conserva per ogni entry i motivi della sua inclusione.

### G.7 PASS / ISSUE workflow

Gli esiti umani sono esclusivamente:

```text
PASS
ISSUE
```

Una review non svolta non ha esito e non può essere interpretata come PASS.

A attesta la curation dettagliata e le verifiche assegnategli. B attesta il controllo indipendente richiesto da G.6. Nessuno dei due deve simulare il ruolo dell’altro.

Ogni issue registra almeno:

```text
issue_id
dataset
stable_key oppure group_key
problem
source_or_curation_to_recheck
reviewer_id
```

Ogni correzione registra:

```text
curation correction
previous value
new value
reason
affected keys
```

Un dato ufficiale non viene alterato per farlo coincidere con un giudizio curatoriale. Un errore di trascrizione viene corretto tornando all’evidenza congelata.

Dopo una correzione il tooling ricalcola selezione, validazione, distribuzioni e campioni. B deve riesaminare:

```text
all selected entries whose review-relevant content changed
all newly selected entries
all newly required sampled or edge-case entries
all entries or groups explicitly implicated by the issue
```

Per i first names, B riesamina anche i confronti generazionali modificati. Per i frammenti modificati, A e B riesaminano le sei combinazioni.

Le approvazioni di entry immutate possono essere riutilizzate per riferimento, purché siano immutati anche peso, gruppo e informazioni rilevanti per la review. Una correzione non impone di ripetere l’intera review di elementi estranei e immutati.

Il PASS finale richiede:

```text
automatic validation: zero errors
required A coverage: complete
required B coverage: complete
open issues: zero
human attestations: bound to current hashes
D1-A hashes: unchanged
```

L’attestazione finale sugli hash non richiede una nuova lettura integrale quando il contenuto già revisionato è identico. Richiede la conferma del collegamento fra contenuto approvato e byte correnti.

Codex può generare report e verificare copertura e hash. Non può assegnare giudizi umani, chiudere autonomamente issue culturali o produrre attestazioni a nome dei reviewer.

### G.8 — Regola di distribuzione

*Sostituisce soltanto la sottosezione «Regola di distribuzione»; gli artifact di provenance già previsti restano invariati.*

D1-B **DEVE registrare e soddisfare tutti gli obblighi applicabili di attribuzione e riuso degli input empirici**.

D1-B **DEVE conservare la provenance della curation originale e le autorizzazioni dei contributori**.

La licenza applicata agli asset curati risultanti è una decisione di distribuzione del repository/progetto. Viene congelata in D1-C oppure nella policy di licensing del progetto; **non viene dedotta automaticamente dalla licenza della fonte**.

In assenza di una decisione già approvata, si registra:

```text
output_license_status = deferred_to_project_policy
```

Questo stato non impedisce curation, tooling, review o congelamento tecnico D1-B. Non costituisce però una decisione di licenza per la successiva distribuzione esterna.

La policy del progetto non può essere usata per omettere obblighi di attribuzione o riuso già accertati per materiale incorporato. Eventuali incompatibilità concrete devono identificare input, obbligo e modalità di utilizzo incompatibile; non si presume un’incompatibilità soltanto perché la licenza finale non è ancora stata scelta.

La rettifica non accerta nuove incompatibilità e non impone CC BY 4.0 all’insieme dei tre dataset.

Resta obbligatorio distinguere contenuti consultati da contenuti copiati o estratti. Non si applicano soglie percentuali arbitrarie per dichiarare automaticamente ammissibile una copia. Licenze e contenuti D1-A rimangono invariati.

### G.9 Execution workflow e significato di readiness

**READY FOR CURATION IMPLEMENTATION non autorizza Codex a produrre e approvare autonomamente i dataset finali.** Richiede la sequenza seguente.

**Stage 1 — Tooling ed evidenze: Codex.** Predisporre tooling deterministico, ledger template, validatori, golden ASCII, controlli D1-A e generazione dei report. Acquisire e congelare soltanto evidenze consentite, secondo C.4 e le fonti già approvate. Eventuali candidati proposti automaticamente restano bozze prive di approvazione.

**Stage 2 — Curation: A, con supporto meccanico di Codex.** A popola e valuta le pool, assegna categorie e annotazioni e risolve l’ammissibilità dei candidati. Codex calcola ranking e selezioni e produce CSV candidati e report preliminari. A controlla tutti i 400 pannelli street. I file restano non approvati.

**Stage 3 — Controllo indipendente: B.** B verifica gli anchor moderni e svolge la review G.6 sui file candidati identificati dai relativi hash. Restituisce `PASS / ISSUE`. Le issue tornano ad A per correzione, con rigenerazione deterministica e riesame secondo G.7.

**Stage 4 — Materializzazione finale e audit: Codex.** Dal ledger approvato, rigenerare i CSV canonici e tutti gli audit. I byte devono coincidere con quelli dei candidati revisionati. Qualsiasi differenza di contenuto torna al percorso di review; non viene assorbita come “pulizia” automatica.

**Stage 5 — Attestazione finale: A e B.** Confermare il PASS sugli hash correnti, la copertura delle rispettive verifiche e l’assenza di issue. Il collegamento agli hash può riutilizzare revisioni già effettuate su contenuto identico.

**Stage 6 — Freeze.** Congelare CSV, ledger, evidenze ammesse, configurazione, audit, review record e hash D1-A verificati. Registrare lo stato della policy di licenza senza iniziare D1-C.

Nessuno stage può sostituire il giudizio umano con punteggi o attestazioni inventati da un LLM.

### H. Open risks e verifica del contratto

| RiskSeverityRequired action                                      |                           |                                                                                                       |
| ---------------------------------------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------------------- |
| Evidenza storica non equivalente a conteggi nazionali per coorte | **ACCEPTABLE LIMITATION** | Applicare C.3 e dichiarare la natura curatoriale dei pesi                                             |
| Anacronismi o differenziazione artificiale delle generazioni     | **REVIEW REQUIRED**       | B esamina tutte le liste selezionate e i confronti C.6                                                |
| Acquisizione degli anchor moderni ancora da completare           | **REVIEW REQUIRED**       | Congelare le dodici viste e verificare le osservazioni; nessuna imputazione per viste mancanti        |
| Cognomi non derivati da una distribuzione nazionale misurata     | **ACCEPTABLE LIMITATION** | Mantenere ORIGINAL CURATED e non presentare i pesi come prevalenze                                    |
| Obblighi degli input o autorizzazioni non documentati            | **REVIEW REQUIRED**       | Completare provenance e autorizzazioni; documentare eventuali incompatibilità concrete                |
| Frammenti innaturali con uno dei sei tipi                        | **REVIEW REQUIRED**       | A controlla tutti i pannelli; B il proprio insieme obbligatorio; correggere senza eccezioni runtime   |
| Lettere non supportate da ASCII Atom v1                          | **ACCEPTABLE LIMITATION** | Registrare le esclusioni senza translitterazioni o grafie artificiali                                 |
| Implementazione, audit e hash finali ancora da attestare         | **REVIEW REQUIRED**       | Eseguire validazione, golden, verifica Unicode 16.0.0 e confronti D1-A sull’implementazione effettiva |

La rettifica ha riscontrato la coerenza di quote, selezioni, cardinalità dei campioni e dei **37 golden** sugli input elencati. Questo non certifica una futura implementazione completa di Unicode 16.0.0.

Non sono stati prodotti o approvati i dataset finali. Le attività umane e di acquisizione ancora da eseguire sono passaggi del workflow, non decisioni curatoriale-strategiche lasciate a Codex.

## 3. Human-effort summary

Le cardinalità seguenti riguardano **una candidata completa**. Le successive correzioni richiedono riesame dei soli elementi e confronti interessati; non esiste un massimo assoluto di iterazioni prima della risoluzione delle issue.

| AttivitàCuratore AReviewer B                 |                                                                                                 |                                                                                             |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| First names storici: otto gruppi             | **960 candidati** valutati                                                                      | **640 righe selezionate** esaminate                                                         |
| Approfondimenti first names storici          | Inclusi nella curation                                                                          | **320 righe di base**, già comprese nelle 640, più edge case interni alla stessa lista      |
| First names moderni: due gruppi              | **240 candidati** valutati                                                                      | **160 righe selezionate** esaminate                                                         |
| Approfondimenti first names moderni          | Inclusi nella curation                                                                          | **80 righe di base**, già comprese nelle 160, più edge case interni alla stessa lista       |
| Cognomi                                      | **750 candidati** valutati                                                                      | **240 righe di base + edge case non già inclusi**, massimo **600 righe distinte**           |
| Frammenti: curation                          | **480 candidati** valutati                                                                      | Nessun secondo scoring della pool                                                           |
| Frammenti: compatibilità                     | **400 pannelli**, equivalenti a **2.400 combinazioni**                                          | **164 pannelli di base**, equivalenti a **984 combinazioni**, più edge case non già inclusi |
| Evidenza moderna                             | Al massimo **120 osservazioni nome/posizione** da acquisire manualmente, se non automatizzabili | Al massimo **120 osservazioni** da confrontare con le dodici viste                          |
| Frequenze annuali da trascrivere manualmente | **0**                                                                                           | **0 verifiche di trascrizioni di frequenze**                                                |
| Secondi punteggi numerici sui candidati      | —                                                                                               | **0**                                                                                       |

Per rendere espliciti anche i limiti superiori dipendenti dal contenuto:

```text
Surname B review:
    240 + s distinct selected rows
    0 <= s <= 360
    maximum = 600 rows

Street B review:
    164 + e distinct selected fragments
    0 <= e <= 236

Street combinations reviewed by B:
    984 + 6e
    maximum = 2,400 combinations
```

Il massimo street di 2.400 per B si verifica soltanto se **tutti** i frammenti risultano appartenere all’insieme obbligatorio, compresi gli ulteriori edge case. Non è più una duplicazione integrale imposta a ogni dataset.

A compila complessivamente **2.430 valutazioni di candidati**. B non le replica: esamina **800 first-name rows**, **240–600 cognomi** e **164–400 pannelli street**. Common, campioni ed edge case vengono deduplicati; una stessa entry non genera più giudizi separati solo perché appartiene a più sottoinsiemi.

## 4. Final readiness

**L1-D1-B CONTRACT RECTIFIED — READY FOR CURATION IMPLEMENTATION**