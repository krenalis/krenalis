# Fase 4B — Confronto indipendente delle architetture di interrogazione Unified

**Stato: B ridotta confermata dal titolare come architettura di riferimento; decisione consolidata dopo la revisione indipendente di Claude.** La sufficienza esecutiva completa della 1.2 non è attestata. I sottocasi non conformi o ancora ambigui restano non attivabili, con gate locali (§§11–14). Nessuna implementazione di produzione è avviata.

**Baseline verificata:** `typed-filter-values@c23777219db49fbf6164b88b4e74ca55a5830f64`. Le specifiche [06 v1.2](06-specifica-semantica-riconciliata-v1.2.md), [07 v1.2](07-ast-canonico-riconciliato-v1.2.md) e [08 v1.2](08-ast-canonico-estratto-v1.2.md) sono normative e approvate. Il [rapporto 13](13-rapporto-revisione-semantica-mondo-chiuso-v1.2.md) ne conserva le decisioni. [11](11-ricognizione-architetturale-filtering-warehouse-fase-4b.md) è una ricognizione storica del codice; [12](12-proposta-architettura-interrogazioni-unified-fase-4b.md) rimane una proposta architetturale non approvata.

## 1. Decisione architetturale consolidata e condizioni di attivazione

È confermata **B ridotta: compilazione semantica condivisa dal Canonical AST validato, generazione SQL e piccoli contratti interni di grana, ownership, compatibilità e dipendenze**. Non sono previsti un piano relazionale autonomo, un optimizer generale o un lowering separato. Le funzioni del compilatore realizzano il significato delle costruzioni; non servono cinque sottosistemi né un interprete separato.

Il contratto consolidato è **risultato completo ed esatto, oppure errore tecnico dell’operazione**. Una singola istruzione SQL è una strategia dimostrata per alcuni frammenti, non un requisito universale né una prova di esecuzione efficiente di tutta la 1.2. Nessun executor ibrido obbligatorio, risultato pubblico parziale o stato tecnico pubblico per profilo viene approvato per implicazione (§7.4).

Il minimo aggiunto rispetto ad A è poter distinguere e controllare ciò che un frammento produce: una riga per occurrence, candidato, tupla selezionata, bucket o profilo; quali owner/chiavi conserva; quali informazioni di incertezza devono ancora essere consumate. Un frammento SQL arbitrario con un generico flag `nullable` non basta per queste verifiche.

Il confronto storico con A resta valido: può emettere lo stesso SQL e avere le medesime ottimizzazioni. Se i suoi helper restituiscono proprio questi contratti sulle relazioni, la distanza da B è soprattutto terminologica. Non occorre costruire un framework aggiuntivo per poter chiamare la soluzione B. La preferenza riguarda **contratti verificabili ai confini dei moduli**, non il numero dei livelli nel diagramma.

C offre vantaggi reali per ispezione e riscritture fisiche, ma aggiunge un linguaggio, una validazione e un passaggio di lowering che non eliminano il lavoro difficile: derivare correttamente selezioni compatibili, deduplicazione, dipendenze e prove ternarie. Non è emerso un requisito attuale che imponga questo costo. Il confronto A/B/C di §4 è conservato come motivazione della scelta; non è riaperto dal consolidamento. Gli algoritmi locali difficili non dimostrano la necessità di C.

**La sufficienza è dimostrata soltanto per i frammenti e le fixture esplicitamente provati.** Non esistono tre compilatori completi A/B/C confrontati sperimentalmente. In particolare, la derivazione generale delle alternative compatibili di E, dei vincoli con valori condivisi e di parte dell’incertezza temporale resta un punto da dimostrare prima dell’approvazione esecutiva. Non è corretto dichiarare risolto quel problema introducendo un tipo `Plan`, né restituendo UNKNOWN più spesso del previsto.

## 2. Metodo e forza delle evidenze

Uso le seguenti qualificazioni nel rapporto:

- **R — Repository:** comportamento letto nel codice della baseline. I rapporti storici servono da indice, non da prova sostitutiva.
- **P — Prototipo:** comportamento effettivamente eseguito in prove isolate; perimetro e comandi storici in §10, nuove verifiche in §14.4.
- **H — Ipotesi/progetto:** contratto o strategia proposta, ancora da implementare o certificare.
- **D — Decisione consolidata:** gli impegni esplicitamente confermati in §§1, 7.4, 12 e 14.1; non rende approvati tutti i dettagli H.

Le prove originarie R/P e i comandi di §10 sono conservati come evidenza storica. Le verifiche mirate di questa esecuzione sono distinte in §14.4. I rapporti [15](15-verifica-fattibilita-architettura-minima-x1-x2.md) e [16](16-analisi-complessita-algoritmica-unified-fase-4b.md) restano integralmente invariati: la nuova analisi registra qui limiti e correzioni, senza attribuire retroattivamente copertura ai loro prototipi.

Letti integralmente i rapporti 11 e 12 e ricontrollati i percorsi necessari alle scelte: preparazione, mapping, query, renderer, scanner, persistenza, listing ed export. Il loro codice applicativo coincide con quello dei commit documentali successivi alla ricognizione: nessuna modifica applicativa è presente nel checkout di partenza.

Le prove utilizzano copie temporanee, overlay Go e un container PostgreSQL nuovo, senza rete e senza mount del repository. Nessuna connessione è stata aperta verso warehouse applicativi. Il container è stato fermato e rimosso. Non sono stati eseguiti SQL remoti su Snowflake: le variabili richieste dal suo tester non sono configurate e non risultano strumenti Snowflake connessi nella sessione. I test locali del package Snowflake e il rendering in overlay **non sono** una certificazione del database remoto.

Le specifiche approvate, 03, 11, 12, 13 e il codice rimangono immutati. Questo consolidamento aggiorna soltanto il contenuto del rapporto 14 e include nel commit documentale dedicato i rapporti 14, 15 e 16, precedentemente non tracciati (§13).

## 3. Infrastruttura esistente: possibilità e adattamenti effettivi

| Evidenza R | Possibilità di riuso | Confine da rispettare |
|---|---|---|
| `core.Filter` ha gruppi e condizioni; le sole varianti implementate sono `*Filter` e `*FilterCondition` | Preservare DTO, 26 operatori e preparazione Record | Il Canonical AST e le nuove varianti Unified sono un progetto normativo, non codice già disponibile |
| `prepareFilter`/`prepareFilterCondition` preparano path e operandi; `prepareFilterGroup` controlla limiti | Riutilizzare la validazione Record | Non convertire tutti i predicati governati in condizioni Record per somiglianza del nome |
| `convertWhere` produce `MultiExpr`/`BaseExpr`, conserva JSON typed e segmenti delle chiavi | Riutilizzare binding delle foglie Record e presenza degli object appiattiti | Non perdere JSON raw/NativeCurrency; non modificare il persistito |
| `RowQuery` ha tabella, colonne, join, WHERE, ordine e OFFSET/LIMIT | Conservare il percorso delle query semplici | Mancano relazioni derivate, proiezioni calcolate, GROUP BY, aggregati, ranking e output ternario proiettato |
| `Expr` è chiusa e implementata da `MultiExpr`/`BaseExpr` | Il renderer è già un produttore di espressioni booleane nullable | Non aggiungere nodi semantici V/E/K/P/T nei driver, né un `RawSQLExpr` pubblico |
| Entrambi i `Query` contano e leggono con due chiamate distinte | Connessioni, scanner e normalizzazione sono riutilizzabili internamente | Il totale conta righe dopo i join e non appartiene necessariamente alla stessa vista della pagina |
| `Counts` esegue un’unica SELECT e valida conteggi fino a `MaxInt32` | Conservare il suo contratto e gli usi attuali | Non accetta una relazione Unified derivata, né conta automaticamente FALSE/NULL o profili distinti |
| Entrambi i driver hanno `RawQuery`, **non esposto da `Warehouse`** | È già disponibile il nucleo per eseguire SQL interno | Restituisce righe grezze e numero di colonne, senza schema risultato o bind parameters nell’attuale firma |
| `QueryReadOnly` valida SQL testuale con policy conservative | Conservare il confine MCP/read-only | Non è il trasporto pronto per SQL interno: PostgreSQL rifiuta `::`, usato dai renderer; Snowflake ha una propria allowlist |
| `newScanner` è privato e usa `Column` per normalizzare i valori | Un nuovo ingresso interno può richiamarlo senza riscriverlo | Il suo `Scan(slice...)` assegna elementi della slice; `RawQuery`/`QueryReadOnly` usano invece lo Scan a puntatori del driver |

Riferimenti verificati: [`core/filters.go:31`](/Users/marco/src/krenalis/core/filters.go:31), [`prepareFilter:502`](/Users/marco/src/krenalis/core/filters.go:502), [`prepareFilterGroup:722`](/Users/marco/src/krenalis/core/filters.go:722), [`convertWhere:58`](/Users/marco/src/krenalis/core/internal/datastore/query.go:58), [`RowQuery:386`](/Users/marco/src/krenalis/warehouses/warehouses.go:386), [`Expr:796`](/Users/marco/src/krenalis/warehouses/warehouses.go:796), [`Warehouse:165`](/Users/marco/src/krenalis/warehouses/warehouses.go:165).

Esecuzione: [`PostgreSQL.Query:17`](/Users/marco/src/krenalis/warehouses/postgresql/query.go:17), [`Snowflake.Query:17`](/Users/marco/src/krenalis/warehouses/snowflake/query.go:17), [`PostgreSQL.Counts:249`](/Users/marco/src/krenalis/warehouses/postgresql/warehouse.go:249), [`Snowflake.Counts:303`](/Users/marco/src/krenalis/warehouses/snowflake/warehouse.go:303), [`RawQuery PG:17`](/Users/marco/src/krenalis/warehouses/postgresql/rawquery.go:17), [`RawQuery SF:15`](/Users/marco/src/krenalis/warehouses/snowflake/rawquery.go:15).

I due accessi PostgreSQL non nascondono una transazione comune: il codice vendorizzato di [`pgxpool.Query:764`](/Users/marco/src/krenalis/vendor/github.com/jackc/pgx/v5/pgxpool/pool.go:764) e [`QueryRow:791`](/Users/marco/src/krenalis/vendor/github.com/jackc/pgx/v5/pgxpool/pool.go:791) acquisisce la connessione a ogni chiamata. `QueryReadOnly` apre una transazione read-only per la propria istruzione, non una sessione riutilizzabile fra chiamate: [`PG:26`](/Users/marco/src/krenalis/warehouses/postgresql/queryreadonly.go:26), [`SF:19`](/Users/marco/src/krenalis/warehouses/snowflake/queryreadonly.go:19), [rifiuto dei cast PG:77](/Users/marco/src/krenalis/warehouses/postgresql/internal/readonlysql/validate.go:77).

### 3.1 Riuso Record senza rendere obbligatoria la qualificazione delle colonne

**P:** il renderer attuale può essere usato su una proiezione con nomi univoci:

```sql
SELECT <predicato Record su "u17"> AS truth
FROM (SELECT <colonna o valore JSON> AS "u17") AS leaf
```

Il compilatore riassocia soltanto il riferimento fisico della foglia a `u17`, conservando tipo, nullability, operatore, operandi preparati, `Keys` e `JSONComparison`. Non riscrive il SQL prodotto. Il renderer vede ancora un identificatore semplice valido. In presenza di più input si proiettano slot diversi prima di valutare la foglia: un JSON key segment contenente punti rimane una chiave letterale, non un identificatore SQL qualificato.

Un overlay, senza scritture nei package originali, ha chiamato i **renderer reali** di entrambi i driver: 28 query generate per dialetto; quelle PostgreSQL sono state eseguite con esiti booleani/NULL verificati. `Column.Name="p.properties"` è stato anche verificato come rifiutato. La prova riguarda quattro operatori JSON (`is`, `is not`, membership positiva e negativa) e sette valori; non l’intero catalogo in ogni contesto.

**H:** serve rendere disponibile il rendering di foglia al generatore comune attraverso un adattatore interno. Non è necessario cambiare ora tutti i renderer per supportare colonne qualificate. Un supporto strutturato `RelationRef + ColumnName` rimane utile se il profiling dimostra un costo delle proiezioni, ma non è un prerequisito già dimostrato. Gli alias devono sempre passare dal quoting del rispettivo dialetto, coerentemente con il renderer che li consuma. Nessuna concatenazione `alias + "." + nome` nel vecchio `Column.Name`.

Riferimenti R: [`renderExpr PG:21`](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:21), [validazione nome PG:57](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:57), [`renderExpr SF:32`](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:32), [serializzatore PG:561](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:561), [serializzatore SF:674](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:674).

### 3.2 Estensioni circoscritte, indipendenti da A/B/C

| Intervento possibile | Utilità | Necessità rispetto a Unified |
|---|---|---|
| Adattatore interno per rendering Record su slot risolti | Riuso dei 26 operatori come valori ternari | Necessario un punto di accesso, non una nuova implementazione del catalogo |
| Ingresso per statement compilato con schema risultato e argomenti opzionali | Riuso pool/DB, scanner, cancellazione ed errori | Necessario un percorso eseguibile; nome/firma e collocazione restano da approvare |
| Riferimenti fisici qualificati e alias nelle sole nuove query | Composizione e self-join | Soddisfacibili inizialmente con proiezioni e riferimenti interni; non impongono cambi al DTO Record |
| Ordinamento per colonna con direzione e politica NULL esplicite | Listing deterministico e futuro seek | Utile anche a RowQuery, ma può nascere nel nuovo percorso senza migrare quello Record |
| Evitare il totale quando non richiesto da `Store.query` | Riduce lavoro inutile già individuato | Correzione indipendente; non risolve count/pagina Unified |
| Proiezioni calcolate/CTE/ranking aggiunti tutti a RowQuery/Expr | Potrebbero esprimere Unified | Trasformerebbero quei tipi in un altro query language: nessuna necessità di farlo per riuso nominale |

Il nuovo ingresso non deve ricevere SQL dall’utente. Può essere un metodo warehouse oppure un piccolo adattatore interno sopra le primitive concrete già presenti. Aggiungere `RawQuery` all’interfaccia senza specificare normalizzazione, schema e parametri non risolve il contratto. Una facciata Profiles condivisa può inizialmente essere una funzione del servizio/datastore, senza un nuovo oggetto pubblico o un’interfaccia per ogni famiglia.

### 3.3 Differenze fisiche da non nascondere nel riuso

**R:** gli identificatori quotati restano case-sensitive nel renderer PostgreSQL; quello Snowflake li converte in maiuscolo. I confronti ordinati JSON String usano rispettivamente collazioni `C` e `utf8`. I literal datetime vengono resi in UTC con sei cifre frazionarie in PostgreSQL e nove in Snowflake. Questi dettagli rimangono nel codice dei dialetti: [`quoteIdent PG:12`](/Users/marco/src/krenalis/warehouses/postgresql/quote.go:12), [`quoteIdent SF:63`](/Users/marco/src/krenalis/warehouses/snowflake/quote.go:63), [`renderJSONComparison PG:249`](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:249), [`renderJSONComparison SF:278`](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:278), [tempi PG:583](/Users/marco/src/krenalis/warehouses/postgresql/renderexpr.go:583), [tempi SF:696](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:696).

Le prove non attestano che ogni confronto String ordinario, Float/NaN, timestamp o quantità mantenga lo stesso significato con ogni configurazione dei due warehouse. I binding delle nuove operazioni governate devono dichiarare rappresentazione e precisione sufficienti; un timestamp non acquista ordine affidabile grazie a un cast. Le questioni già aperte sulla portabilità Record richiedono caratterizzazione, senza correzioni retroattive, nuove conversioni implicite o requisiti di registry aggiunti ai normali filtri Record. L’architettura deve rappresentare tutti i 26 operatori, non evitarne alcuni per semplificare il backend.

Per esempio, se T ha precisione maggiore di quella codificata nel literal, troncarlo può cambiare il risultato di `event_timestamp < T` sul bordo. Fissare lo stesso T è necessario, ma non dimostra che entrambi gli encoding realizzino lo stesso confronto. È una verifica di X5 per i nuovi predicati temporali, non un’autorizzazione a modificare gli operatori Record esistenti.

## 4. Confronto delle alternative sui medesimi criteri

Le lettere qui sono quelle dell’incarico: **non** quelle della vecchia tabella del rapporto 12.

| Criterio | A — Canonical AST → SQL diretto | B — Moduli e relazioni tipizzate | C — Piano intermedio tipizzato |
|---|---|---|---|
| Correttezza semantica | Possibile; tutti gli obblighi sono negli helper | Stessi algoritmi; grana/owner/dipendenze diventano contratti fra moduli | Stessi algoritmi prima/durante il lowering; la tipizzazione relazionale da sola non li dimostra |
| Test | Ottimi test di esiti; più difficile ispezionare contratti se si restituiscono solo stringhe | Test di esiti più test dei confini di relazione | Aggiunge test di nodi, validatore e lowering; utile se quei passaggi hanno responsabilità effettive |
| Riuso attuale | Renderer, binding, codec e trasporto adattato | Uguale ad A, più relazioni condivise fra listing/count/export e moduli | Uguale, più eventuali operatori fisici riutilizzati |
| Nuovi contratti | Contesto/dialetto, composizione e output eseguibile; i confini semantici restano comunque necessari | Quelli di A più schemi/grane nominali sulle relazioni richieste | Quelli di B più algebra fisica, validatore, riferimenti fra nodi e lowering |
| Duplicazione PG/SF | Bassa con generatore comune; alta con due generatori per primitiva | Bassa se i moduli generano una struttura comune | Bassa solo se anche il rendering della struttura è comune; due interpreti del piano la reintroducono |
| SQL leggibile | Possibile con CTE nominate e helper piccoli | Nomi e contratti favoriscono una corrispondenza locale | Possibile, ma serve un generatore che conservi tale corrispondenza |
| Prestazioni | Può produrre lo stesso SQL di B/C | Può produrre lo stesso SQL di A/C | Il piano non rende automaticamente migliori le scelte dell’optimizer del database |
| Scalabilità del compilatore | Rischio di espansione del testo/alias se il codice è ad hoc | Dipendenze e riuso controllabili senza optimizer generale | Migliore spazio per più strategie fisiche, al costo di altri passaggi |
| Manutenibilità | Minima infrastruttura iniziale; errori di grana se le convenzioni restano implicite | Compromesso favorevole per cinque famiglie chiuse e due dialetti | Maggior costo per estensioni che devono attraversare AST, compilatore, IR e SQL |
| Evoluzione | Modifica locale del relativo helper | Modifica locale più verifica delle firme prodotte/consumate | Utile se cambiano backend/strategie; più combinazioni da mantenere |
| Sicurezza | Costruzione controllata necessaria; testo SQL non significa automaticamente insicurezza | I tipi aiutano, ma non sostituiscono quoting e validazione dei binding | Il piano aiuta, ma non garantisce sicurezza del renderer o dei risultati |

Non assegno punteggi numerici o vantaggi di latenza non misurati. I prototipi provano alcune traduzioni, non il costo comparato di tre implementazioni né la scalabilità su storie reali. A e B possono coincidere nel codice; introdurre un piano solo per rendere più distinta la classificazione non sarebbe un vantaggio.

C può accorpare o sostituire alcuni wrapper di B, non deve necessariamente accumularli tutti. Il costo aggiuntivo da valutare è quello effettivo dell’algebra fisica, della sua validazione e del lowering; i contratti di grana e dipendenza rimangono necessari in qualunque rappresentazione.

## 5. Contratto minimo delle relazioni e responsabilità software

### 5.1 Responsabilità necessarie, senza un componente per ciascuna

1. Validare contesto, shape, owner, tipi e riferimenti; riutilizzare la preparazione Record.
2. Associare i riferimenti alle fonti/colonne e allo stesso T/calendario.
3. Compilare il significato della costruzione, conservando grana, dipendenze e guardie.
4. Comporre SQL, alias e parametri; rendere soltanto le differenze di dialetto.
5. Eseguire, normalizzare e consumare i risultati per listing/count/export.

Sono responsabilità distinte, non cinque servizi o cinque interfacce obbligatori. Per iniziare bastano funzioni/moduli comuni, un compositore privato e gli adattatori dei due driver. Il compilatore può scegliere una strategia locale mentre emette il modulo; non serve un `Planner` autonomo senza alternative fisiche effettivamente implementate.

### 5.2 Forme di risultato interne utili

Schema concettuale H, **non** nuovi costruttori del Canonical AST o firme Go già approvate:

| Relazione | Grana / chiave | Informazione che non deve sparire prima del consumo |
|---|---|---|
| Occurrence | `(profile, occurrence/grain identity)` | Qualificazione ternaria, valori originali, ordine/tempo e riferimenti ai fatti condivisi |
| Selected tuple | `(profile, owner, alternative, slot)` con identità della tupla | Compatibilità congiunta dei selezionati e fatti comuni che condizionano selezione/verifica |
| Candidate body | `(profile, owner, candidate identity)` | Appartenenza alla base e risultato del corpo distinti; candidati incerti e guardie |
| Path witness | `(profile, owner, tuple, common key)` nel blocco | Stessa chiave lungo tutto il blocco, occurrence distinte, vincoli temporali e negativi della stessa prova |
| Bucket | `(profile, owner, bucket ordinal)` | Tutti gli M bucket, estremi e risultato del recognizer anche sul vuoto |
| Profile truth | `(profile)` | Una riga per profilo della platea, TRUE/FALSE/NULL e ragioni richieste, senza moltiplicazioni da export |

I nomi della tabella descrivono grane e informazioni, **non impongono sei contratti o un numero prefissato di strutture Go**. Pochi wrapper o un unico tipo con costruttori privati possono raggruppare riferimento SQL, profilo, owner, identity/dedup, grana, schema e significato delle colonne, precisione, fatti condivisi, compatibilità e punto di chiusura. Si conserva ogni informazione necessaria al consumer anche quando più forme usano lo stesso tipo. Non occorre un AST traversabile né rappresentare ogni operazione come `ProjectNode/JoinNode/AggregateNode`.

Profilo e owner devono essere controllati anche nei join e nelle riduzioni: metadati corretti non provano che il producer abbia emesso SQL corretto. Servono test multiprofilo, dei join e delle invarianti interne; compatibilità delle selezioni e dipendenza part/whole non diventano facoltative perché non hanno una struct dedicata.

Un compositore limitato registra CTE e dipendenze, assegna nomi sicuri, risolve colonne/parametri e costruisce il SELECT finale. Le dipendenze devono essere acicliche; una CTE non deve consumare l’owner sbagliato o un risultato a grana diversa senza una riduzione esplicita del modulo. Non aggiungerei serializzazione, canonical equality o un hash pubblico di queste relazioni. `ProfileTruthRelation` non promette che il database materializzi il risultato.

### 5.3 L’informazione ternaria non è sempre un solo flag

Una colonna SQL nullable basta **dopo** che una foglia/owner ha consumato le proprie dipendenze. Prima può non bastare. Due righe con qualificazione UNKNOWN possono dipendere dallo stesso valore corrente del profilo; una identity multivalore deve restare comune all’intero blocco; un numerator share non è indipendente dal whole.

Gli input canonici risolti conservano già il riferimento al medesimo fatto. Il modulo deve usarlo finché valuta la costruzione; non duplicarlo come incognite indipendenti. Le relazioni di alternative sono un possibile veicolo locale di questo vincolo, non un linguaggio generale di mondi possibili o un nuovo registry. Nelle prove originarie di §10 le alternative sono **enumerate dalle fixture**. Il rapporto 15 ne deriva alcune da input tipizzati, ma non offre un algoritmo generale e contiene il difetto E registrato in §§6.2 e 14.4.

Il punto di chiusura cambia per costruzione: E conserva fatti condivisi anche se letti soltanto nel verify, fino al test sulla selezione compatibile; P conserva fatti condivisi **fra tuple alternative** fino alla decisione sull’esistenza del percorso per profilo, oltre a tupla e chiave comuni all’interno di ogni prova; share conserva part/whole e i fatti comuni al test; K **dopo la valutazione individuale dei candidati** applica N2 e non propaga inferenze globali fra candidati. I confini non completamente definiti per basi K correlate sono registrati in §14.3, senza attribuire una semantica nuova al prototipo. L’AND/OR principale conserva la logica ternaria normativa, senza inferenze simboliche fra rule sorelle. Un optimizer che conservasse globalmente tutte le dipendenze e “migliorasse” Some della fixture N2 a MATCH sarebbe non conforme.

### 5.4 Adattatori e output eseguibile

Il contratto eseguibile minimo proposto contiene SQL generato internamente, argomenti tipizzati se utilizzati e schema delle colonne risultato. T, calendario e binding risolti sono input della compilazione; non si inventano qui generazioni o watermark. Non occorre passare al driver V/E/K/P/T, candidate quantifiers o meaningRevision da interpretare.

La struttura comune può usare costrutti già presenti in entrambi i driver, per esempio `COUNT(CASE WHEN ... THEN 1 END)` per distinguere gli stati. Non si sostituiscono AND/OR o All/None ternari con aggregati booleani che ignorano i NULL. Le guardie restano associate alla costruzione che le possiede e gli errori rimangono esterni ai tre valori semantici.

Agli adattatori restano quoting, literal/parametri esatti, accesso JSON, rappresentazione dei tipi, operazioni temporali/fisiche nominate e pool/scanner/errori. Il modulo comune decide **quando** una valuta ignota incide su un’assenza, non l’adattatore numerico o JSON. I renderer Record attuali continuano a interpretare il proprio catalogo già stabilito: il compilatore li tratta come foglie con contratto noto, senza riscriverli in un secondo repertorio.

Il disegno dei package deve rispettare il confine `core/internal`: il driver non deve importare il modello persistito interno di core. Si possono passare `warehouses.Expr` preparati e schemi risolti agli adattatori; i moduli semantici restano sopra il warehouse. La collocazione e la firma del nuovo ingresso sono una decisione d’implementazione ancora aperta, non una nuova API pubblica definita da questo studio.

## 6. Prove di sufficienza sulle famiglie normative

Le relazioni seguenti sono descrizioni fisiche H, con risultati P dove indicato. Non espongono nuove funzionalità al linguaggio. Le incertezze si riferiscono a dati registrati esistenti: il mondo chiuso non aggiunge eventi mancanti dal mondo esterno. Conserviamo deadline, contratti autorevoli/CDE e tutti i limiti 1.2.

### 6.1 V — Vuoto, proprietà ignote, valuta e precisione

**Relazioni:** platea Profiles a grana profilo; occurrence normalizzate alla grana dichiarata; classificazione del predicato prima della riduzione; riduzione per profilo; terminale con guardie. La platea deve alimentare il risultato anche senza eventi. Un `LEFT JOIN` produce una riga segnaposto: non va contata come occurrence UNKNOWN.

Nelle fixture con appartenenza semplice si calcolano `lo=count(predicato TRUE)` e `hi=count(predicato non FALSE)` **sulle sole occurrence esistenti**. Assenza: FALSE se `lo>0`, TRUE se `hi=0`, NULL altrimenti. Count/sum vuoti valgono zero nella propria unità; media/min/max conservano le guardie. Non basta usare `NOT EXISTS (... WHERE predicato)` perché SQL WHERE eliminerebbe proprio gli UNKNOWN pertinenti.

| Dati / richiesta | Esito P PostgreSQL |
|---|---|
| Nessun Purchase nel repertorio configurato | `lo=hi=0`, assenza TRUE |
| Un Purchase con valuta governata ignota | `[0,1]`, assenza NULL |
| Un Purchase EUR certo | `[1,1]`, assenza FALSE |
| JSON raw `currency:null` confrontato con String EUR | Predicato FALSE, assenza TRUE |
| Stesso null interpretato dal contratto NativeCurrency | Predicato NULL, assenza NULL |
| Somma `9007199254740993.01 + 0.02` | Esatta `9007199254740993.03`, senza passaggio a float |
| Importi 10 e ignoto, con dominio non negativo certificato | Somma esatta non determinata; il 10 noto può già provare `sum>=5` |
| Scienza 0, Arte 10, Tecnologia 10, una categoria ignota | “Scienza unica prevalente” FALSE, come N3 |

**Precisione:** la condivisione del SQL non rende identici i domini numerici dei database. Il codice dichiara operandi Decimal fino a precisione 76/scala 37, e Snowflake ha un renderer JSON numerico specifico; lo schema ordinario Snowflake usa invece `NUMBER(p,s)`. Non si può estendere a `SUM` una garanzia ottenuta per un confronto letterale. Servono codec esatti e limiti/overflow espliciti per ciascuna riduzione e tipo; mai conversioni generiche in float64. Un overflow di esecuzione è un errore tecnico, non UNKNOWN.

Fonti R: [`BaseExpr:833`](/Users/marco/src/krenalis/warehouses/warehouses.go:833), [`renderJSONNumberComparison SF:631`](/Users/marco/src/krenalis/warehouses/snowflake/renderexpr.go:631), [`typeToSnowflakeType:283`](/Users/marco/src/krenalis/warehouses/snowflake/alter_schema.go:283), [`typeToPostgresType:283`](/Users/marco/src/krenalis/warehouses/postgresql/alter_schema.go:283). Il limite di 38 cifre di NUMBER è documentato da [Snowflake](https://docs.snowflake.com/en/sql-reference/data-types-numeric); la prova PostgreSQL non certifica somme Snowflake con gli stessi valori limite.

**Emendamento V verificato:** contributo certo 100 e opzionali indipendenti 30, 50, 80 danno somme raggiungibili `{100,130,150,180,210,230,260}`. I due modi di ottenere 180 non producono una nuova somma. Le prove nuove (§14.4) danno:

| Test sulla stessa somma V | Esito esatto | Perché |
|---|---|---|
| `=190` | NO_MATCH | Nessuna somma compatibile è 190 |
| `≠190` | MATCH | Tutte le somme compatibili sono diverse da 190 |
| `≠180` | UNKNOWN | 180 è raggiungibile, ma non obbligatorio |
| `[185,195]`, estremi inclusi | NO_MATCH | L’intervallo non contiene somme raggiungibili |

I limiti `[100,260]` da soli non decidono i primi due test né l’intervallo. Il rischio combinatorio del rapporto 16 **non è esclusivo di WeightedShare**: riguarda anche riduzioni additive e terminali pertinenti. `Direct(Sum,=c)` contiene già raggiungibilità di somme; `Average=c` può richiedere `Σ(amount−c)=0` sulla stessa popolazione con guardia non-vuota; `Difference` di riduzioni additive deve preservare i fatti comuni ai due operandi. Queste sono trasformazioni interne condizionate da tipi, unità, precisione, guardie e compatibilità, non nuove formule pubbliche o una prova che ogni richiesta di media/differenza sia difficile. Il riuso di DP, bitset o MITM è ammesso soltanto nel dominio e nelle precondizioni dimostrate dal relativo algoritmo (§9.4).

Per distinct, media, distribuzioni, ratio e pesi servono riduzioni specifiche, non un’unica aritmetica d’intervalli. Part/whole e valori condivisi non devono diventare indipendenti; proprietà/grana incerte non autorizzano deduplicazione euristica o `SUM` che ignora i null. Le prove V sopra non dimostrano un algoritmo generale per tutte queste combinazioni.

Riferimenti normativi: 06 §§10, 19.2–19.4, 28.19–28.20 e 28.25; 07 §§4.1.2, 7.

### 6.2 E — Selezioni compatibili, parità e fatti condivisi

**Relazioni:** base qualificata con evidenza ancora leggibile; candidati/tuple alle posizioni ammesse; compatibilità congiunta; verifica della stessa tupla; chiusura dell’owner in una riga ternaria per profilo. La grana intermedia non è genericamente “un evento possibile”: per Last/Previous e latest N è una selezione congiunta di più occurrence.

`ROW_NUMBER` è sufficiente nel percorso rapido con appartenenza e ordine affidabili. Non è un interprete generale di E: `ORDER BY timestamp,id` risolve artificialmente una parità, mentre filtrare `qualificazione IS TRUE` elimina possibili selezionati. Le selezioni di primo/ultimo di P e i pivot/opportunità K devono usare lo stesso nucleo corretto, con le rispettive firme e basi.

**P, casi eseguiti:**

- Purchase EUR score 150, poi Purchase USD score 50: ultimo assoluto `score>100` FALSE; ultimo EUR TRUE.
- Vecchia occurrence certamente qualificante con body TRUE; ultima qualificante solo possibile con body FALSE: esito NULL. Se entrambi i body sono TRUE, esito TRUE.
- Due occurrence a timestamp pari senza ordine affidabile: body entrambi TRUE → TRUE; body diversi → NULL. Il prototipo considera entrambi gli ordini compatibili, non sceglie un ID come ordine reale.

**Il controesempio che un semplice `certain/possible` non risolve.** Il contratto garantisce che una proprietà governata corrente `preferredCategory` esista e abbia esattamente un valore nel dominio chiuso `{A,B}`, ma non ne rende noto il valore. La base di selezione confronta la categoria di ogni evento con quella stessa proprietà (`EqualCurrentProfile`, già previsto dal 07 §4.1). Le quattro occurrence sono ordinate e hanno:

| Occurrence | Categoria | Score |
|---|---|---:|
| e1 | A | 90 |
| e2 | B | 140 |
| e3 | A | 100 |
| e4 | B | 150 |

Richiesta: differenza fra ultimo e precedente score almeno 10. Con A si seleziona `(e3,e1)`; con B `(e4,e2)`. Entrambe le selezioni danno 10: **TRUE**. Incrociare ultimo possibile A con precedente possibile B inventa `(e3,e2)`, differenza −40, e può produrre un falso UNKNOWN. Anche una base con un evento A e uno B non può essere vuota sotto queste premesse: classificare separatamente entrambi come UNKNOWN perde questa garanzia.

**P:** il prototipo SQL mantiene `context=A/B` durante selezione e pairing; restituisce TRUE e identifica una coppia falsa aggiunta dal prodotto cartesiano indipendente. È la dimostrazione di un difetto del modello ridotto, non la prova che qualsiasi incertezza si risolva enumerando due valori.

**H, cosa occorre al modulo E:** mantenere il riferimento al fatto condiviso e produrre soltanto selezioni congiuntamente compatibili, comprese l’eventuale assenza della posizione e le guardie richieste. Per un dominio finito semplice può bastare una relazione locale di alternative; per valori ordinati o altri dati condivisi occorre verificare un algoritmo di compatibilità specifico. Non si propone un risolutore generale di formule o un rafforzamento dell’AND/OR normativo. L’enumerazione esaustiva dei completamenti è utile come oracolo dei test, non come strategia di produzione già scelta.

Una selezione può essere ambigua nell’identità ma determinata nell’esito. Non basta fare `EXISTS` di una selezione favorevole; il fallimento del body non consente fallback. `LatestN` richiede N occurrence e il preset sugli intervalli mantiene gli stessi tre eventi e le guardie su entrambe le durate. Nessun piano A/B/C può ignorare questi requisiti.

**Emendamento E verificato:** due occurrence certamente selezionate leggono lo stesso fatto booleano governato ignoto `g` solo nel verify; il test chiede esattamente un body vero. I conteggi compatibili sono 0 e 2, quindi NO_MATCH. Il prototipo storico 15 accetta il modello `flag_fact="g"` e produce UNKNOWN sia nel compilatore sia nell’oracolo: la correlazione non viene conservata in questo percorso. Non è un caso conforme solo perché i due programmi concordano. Un esperimento isolato aggiunge un controllo di ammissione su qualificazione **e** verify e rifiuta il sottocaso con `PrototypeUnsupported`, senza inventare un risultato. Il supporto effettivo richiede ancora l’algoritmo e un oracolo indipendente (§14.4). I sorgenti storici incorporati nel 15 restano invariati.

Riferimenti normativi: 06 §§11, 19.4; 07 §§4.1, 8, 9.1. **Limite non dimostrato:** traduzione generale ed efficiente delle alternative con valori/qualificazioni condivisi, N fino a 10, ordine parziale e scope relativi.

### 6.3 K — Base candidata, appartenenza incerta e N2

**Relazioni:** base distinta per candidate identity; membership separata dal body; scope same-key/opportunità/contratto; una riga di risultato individuale per candidato; quantificazione per profilo. Refund duplicati non moltiplicano gli ordini, e un `JOIN ... key=...` non deve nascondere Refund registrati con chiave ignota.

Con base determinata, il modulo applica esattamente N2: `D=S+F+U`, inviluppo `[S,S+U]`. Some, None, cardinalità, All e share derivano da questi risultati e dalle loro guardie; non dalla fattibilità globale dei conteggi al loro interno. Per All un F certo confuta; per None un S certo confuta; D certamente zero fa fallire All/share non vacui. Un body UNKNOWN resta nel denominatore.

**P:** due ordini O1/O2 e un solo Refund con orderId ignoto danno due body UNKNOWN, `S=F=0,U=2`. Some/None/All risultano NULL e la share ha limiti 0 e 1. La prova globale “un solo rimborso univalore non può rimborsare due ordini” **non viene utilizzata**. Questa è una regola normativa, non un’ottimizzazione facoltativa.

**P:** due Refund certi dello stesso O1 e nessuno di O2: D=2, un ordine senza rimborsi, share=1/2. Contare le righe del join darebbe una grana sbagliata.

**Base incerta:** se O1 è certamente candidato e O2 è soltanto possibile, non si può usare automaticamente D=2. Il contratto della relazione mantiene separati `membership` e `body`. Il prototipo considera O1 con body TRUE e O2 possibile con body FALSE: All e share al 100% sono NULL. Non è una nuova inferenza fra body K: è conservazione dell’incertezza della base e delle guardie/denominatore.

Una proposta di limiti `D_lo/D_hi` può essere utile, ma **non è una dimostrazione di sufficienza** per identity/deduplicazione incerte e share. Servono regole specifiche che preservino la stessa base nel numeratore e nel denominatore. Il caso con un candidato certo e un possibile, entrambi con body certamente TRUE, deve poter conservare la quota 1 quando il minimo è già soddisfatto; non inventare un’incertezza del body.

Le forme authoritative e CDE mantengono inventari, lifecycle e garanzie proprie. Il mondo chiuso degli eventi non rende completo un inventario non determinato; una capability non disponibile è distinta dal suo risultato sul profilo.

I casi K04/K16/K18 del 15 sono riesaminati in §14.3: distinguere la conservazione della base dalle inferenze vietate sui body, senza considerare dimostrati tutti i casi in cui lo stesso fatto governa entrambi.

Riferimenti normativi: 06 §§12.3–12.6, 15–16, 19.4, 28.24; 07 §9. **Limite non dimostrato:** gestione completa di basi con identity/membership incerte e di tutti i contratti oggetto.

### 6.4 Share e dipendenza part/whole

Il whole è costruito una volta e la part qualifica **quelle stesse righe**, con gli stessi pesi quando previsti. Due righe certe nel numeratore e una nel denominatore non possono provenire da completamenti incompatibili.

**P:** una occurrence certa e una possibile, entrambe certamente nella part quando appartengono al whole. Le sole basi compatibili sono `{e1}` e `{e1,e2}`: quota sempre 1 e minimo soddisfatto. Il SQL conserva questa dipendenza e restituisce TRUE per share=1. Dividere limiti indipendenti `[1,2]/[1,2]` darebbe `[1/2,2]` e perderebbe il risultato; il fatto che l’estremo 2 violi una proprietà della share è già un segnale della relazione perduta.

Questa dipendenza obbligatoria non autorizza la prova globale vietata da N2 fra body K distinti. Non è previsto un unico operatore fisico `UnknownBounds` che si comporti allo stesso modo per V, K, E, P e T.

Con due occurrence certe, part certa per una e incerta per l’altra, `V.FrequencyShare=3/4` è NO_MATCH: le quote compatibili sono `{1/2,1}`. La tabella di 06 §19.4 ne dà un limite, non afferma che tutti i reali nell’intervallo siano raggiungibili. Il diverso inviluppo composizionale richiesto a `K.Share` su base determinata non si trasferisce a V. Il confronto normativo e il caso K18 sono esplicitati in §14.3.

### 6.5 P — Tupla intera, chiave unica e assenze

**Relazioni:** basi profile-scoped di ciascun passo; selezione First/Last, se richiesta, prima della correlazione; espansione delle identity multivalore dichiarate; tupla di 2–5 occurrence distinte; stesso valore di key nel solo blocco; vincoli di ordine/durata e scope negativi riferiti agli slot autorizzati; chiusura esistenziale a grana profilo.

**P:** i join finiti rappresentano tutti i percorsi di lunghezza 2, 3, 4 e 5 provati, senza ricorsione. Un’implementazione non deve cercare solo un percorso greedy né restituire il numero delle combinazioni come membership.

**P, identity multivalore:** A=`{x,y}`, B=`{y,z}`, C=`{x,z}` hanno intersezioni adiacenti non vuote, ma nessuna key comune all’intero blocco. SQL con la stessa colonna `k` lungo tutti i join restituisce FALSE; i due controlli pairwise separati restituiscono TRUE e sono insufficienti. Aggiungendo y a C, il percorso diventa TRUE.

La relazione espansa ha grana `(profile, occurrence, key value)`; il witness conserva il valore scelto anche nelle assenze keyed. I passi esterni al blocco non lo ereditano. Con key ignote servono compatibilità e vincoli di namespace/insiemi noti: `NULL=NULL` non è una key comune, e un segnaposto globale “UNKNOWN_KEY” non può diventare un testimone certo. Un’intersezione certamente vuota fra key note può già confutare una tupla anche se altri passi hanno key ignote.

**P:** per una tupla con estremi a 0 e 10, un errore registrato che può stare a 5 oppure a 15 rende l’assenza intermedia NULL. Per due azioni allo stesso timestamp senza ordine affidabile, la fixture con le due precedenze alternative dà NULL al percorso direzionale. Le alternative sono fornite dalla fixture: non certificano un modello generale di simultaneità/ordine parziale (§14.3).

Un `NOT EXISTS` è lecito solo dopo aver stabilito che non esiste nemmeno un evento registrato possibilmente pertinente allo scope negativo. Le assenze devono usare gli stessi estremi/key della tupla. Per il test complessivo, una tupla certa è **sufficiente**, ma non necessaria, per MATCH. Occorre decidere l’esistenza del percorso sulle alternative congiuntamente compatibili ammesse dall’evidenza: se tale esistenza è certa, MATCH; se è certamente esclusa, NO_MATCH; se può cambiare, UNKNOWN. Non basta chiudere ogni tupla in un flag ternario e farne OR quando uno stesso fatto ignoto determina tuple diverse.

**Controesempio verificato:** `a=Export` alle 10:00, `x` con funzionalità governata ignota alle 10:40, `d=Import` alle 11:30; richiesta Export seguito da funzionalità diversa da Export entro 60 minuti. `(a,x)` vale se `x≠Export`, `(x,d)` se `x=Export`; `(a,d)` è fuori tempo. Esiste una coppia valida in entrambi i casi, quindi MATCH, pur senza una stessa coppia certamente valida. La partizione Export/non-Export è esaustiva del dominio dichiarato del fatto, non una conversione di un JSON raw mancante in un valore. La chiusura avviene **dopo l’esistenza fra tutte le tuple pertinenti del medesimo P**, conservando quel fatto fino a quel punto. È un requisito del gate X3, non un’inferenza fra rule sorelle né un’eccezione a N2.

Riferimenti normativi: 06 §§13, 19.4; 07 §12. **Limite non dimostrato:** produzione completa delle tuple ammissibili con selettori ambigui, key parzialmente note, ordine incerto e più scope negativi condivisi; controllo dei costi nel caso peggiore.

### 6.6 T — Bucket, tempo ignoto, DST e MaximumGap

**Relazioni:** serie calendar esplicita e finita a grana bucket; associazione delle occurrence ai giorni/bucket con evidenza di appartenenza; recognizer per bucket; cardinalità QPC per profilo. I quattro bucket nascono prima del join agli eventi e non spariscono se vuoti. La conta finale usa S/F/U dei recognizer individuali, senza inferenze simboliche fra bucket, come 06 §14.7.

**P:** “almeno due giorni in almeno tre delle quattro settimane” restituisce TRUE per `[2,2,2,0]`, FALSE per `[2,2,1,0]`, NULL per due settimane certe, una con 1–2 giorni possibili e una vuota. Tutte le quattro settimane restano presenti. Un solo evento di giorno ignoto non diventa sette giorni distinti: nel caso provato il massimo è uno, e la soglia due è FALSE.

Questo mette in evidenza un vincolo: espandere le possibili assegnazioni evento→giorno e fare `COUNT(DISTINCT possible_day)` sovrastima giorni ottenibili simultaneamente. Un’upper bound deve rispettare la singola occurrence e i fatti condivisi; se occorrono assegnazioni compatibili non può bastare il solo join espanso. Il limite `known_days + unknown_occurrences`, limitato ai giorni del bucket, è usato nel prototipo soltanto nel caso dichiarato con timestamp ignoti liberamente collocabili e grana determinata.

**P, DST:** PostgreSQL restituisce 23 ore per il giorno locale 8 marzo 2026 e 25 per il 1 novembre 2026 in `America/New_York`. Giorni/settimane si definiscono con bordi calendar nel fuso dichiarato; burst elapsed e ritardi restano durate trascorse. Non si usa `bucket_start + 24h` come definizione generale di giorno. Snowflake documenta che molte operazioni settimanali dipendono da [WEEK_START](https://docs.snowflake.com/en/sql-reference/functions-date-time): il dialetto deve imporre il calendario normativo, non affidarsi alla sessione casuale.

**N1:** MaximumGap riceve soltanto periodi con entrambi gli estremi finiti previsti dalla forma. FullRelevantHistory da solo è invalidità di configurazione prima del SQL. AvailableHistory è valido con intervallo dichiarato completo di estremi; un estremo previsto ma ignoto sul profilo conserva la possibile incertezza semantica. Nessun inizio ricavato dal primo evento o dalla retention.

**P, gap in giorni trascorsi:** nel periodo `[0,14)` senza attività il gap è 14 e il limite 7 è FALSE; un’attività a 7 dà gap massimo 7, TRUE; a 6 dà massimo 8, FALSE. Il ciclo dinamico inizialmente vuoto dà gap 0. Il SQL include entrambi i bordi come punti di calcolo, senza trasformare il bordo finale in un’attività nel periodo, e usa le distanze fra punti consecutivi. Con tempo/qualificazione ignoti non è lecito applicare semplicemente LAG ai soli eventi noti: le evidenze possibili possono interrompere il gap quando decisive. Il prototipo non dimostra questo caso generale.

**Esempio analitico, non eseguito:** in un periodo di 16 giorni trascorsi con una sola occurrence di tempo ignoto, il massimo gap non può essere minore di 8 giorni neppure collocandola nel punto migliore. Il test max gap≤7 è quindi NO_MATCH, non UNKNOWN per la sola presenza di quel timestamp ignoto. Le regole del recognizer devono conservare anche queste prove negative; un flag generico “tempo incerto” sarebbe insufficiente.

Burst, streak e QPC mantengono i propri vincoli; non ricevono il divieto globale su FullRelevantHistory. Per QPC max gap gli estremi sono già quelli dei bucket. Ritorni richiedono un predecessore realmente registrato e consecutività, non un bordo di retention o un evento esterno ipotetico.

Riferimenti normativi: 06 §§14, 17, 28.23; 07 §13. **Limite non dimostrato:** riconoscitori temporali completi con timestamp/qualificazioni condivisi ignoti, fuso del profilo ignoto, finestre personali e precisione dei due backend.

### 6.7 Record — Conservazione del contratto installato

Il percorso resta `core.Filter → prepareFilter → state.Where → convertWhere → Expr → renderer`, con `Applies` per i contesti Record Go. La nuova compilazione può consumare le stesse foglie preparate, ma non trasformare il booleano Go FALSE in una diagnostica NO_MATCH: ha perso la distinzione da UNKNOWN.

Devono restare invariati tutti i nomi dei 26 operatori, payload, validazione tipo/cardinalità, operandi esatti, path segmentati, JSON typed/structural, presenza degli object appiattiti e risultati TRUE/FALSE/NULL. I sei test di stato conservano la propria semantica esplicita; non ricevono una guardia NULL uniforme. Il riuso è per significato e codice del renderer, non per sostituzione testuale del SQL.

**R:** `Where.Equal` preserva struttura e ordine; `MarshalWhere` produce il JSON esplicito dei gruppi/condizioni. I confronti decimal/JSON hanno trattamenti specifici. Il nuovo compositore non canonicalizza questi oggetti né li usa come piano fisico persistito. I serializer e change detector restano nel percorso esistente.

Fonti: [`Where.Equal:34`](/Users/marco/src/krenalis/core/internal/state/where.go:34), [`WhereCondition.Equal:106`](/Users/marco/src/krenalis/core/internal/state/where.go:106), [`MarshalWhere:249`](/Users/marco/src/krenalis/core/internal/state/where.go:249), [`Applies:18`](/Users/marco/src/krenalis/core/internal/filters/applies.go:18).

**P:** superati i test locali selezionati di entrambi i renderer, evaluator Go e persistenza/equality; più le prove di proiezione §3.1. Non è una prova lossless del futuro formato Unified o dell’intera composizione con ogni tipo. Il formato pubblico/persistito delle nuove varianti rimane un lavoro successivo separato; non è necessario deciderlo per scegliere fra A/B/C.

### 6.8 La stessa traduzione, espressa nelle tre alternative

| Prova | A: helper dal Canonical AST | B: modulo e contratto relazionale | C: operazioni del piano, dopo la medesima analisi semantica |
|---|---|---|---|
| V vuoto/valuta ignota | Emette platea, foglie, riduzione e CASE | Occurrence→riduzione→ProfileTruth; mantiene null e grana | Scan/LeftJoin/Project/Aggregate/Project |
| V sum/precisione | Helper sum e guardie specializzati | Riduzione con tipo/unità e informazione di validità | Aggregate/Project con codec e tipi esatti; non un SUM generico autosufficiente |
| E ultimo qualificante | Helper di selezione, alternative congiunte e verify | SelectedTuple con owner/fatti comuni; poi chiusura | Ranking/join/proiezioni su relazioni di alternative già correttamente derivate |
| E fatti condivisi | Conserva contesto locale nel generatore | Contratto impedisce pairing fra alternative diverse | Chiavi e predicati di compatibilità fra nodi; l’IR non li inventa |
| K N2 | Helper body per candidato, poi S/F/U | CandidateBody→quantificatore; punto di chiusura esplicito | Group/Aggregate/CASE; divieto di ottimizzazione globale fra candidati |
| K base incerta/share | Helper specifici sulla stessa base | Membership distinta da body; part/whole comune | Relazioni e aggregati correlati, non divisione di due limiti indipendenti |
| P | Helper di catena finita e scope negativi | Witness a grana tupla+key; fatti comuni conservati fra tuple fino alla chiusura del percorso | Join/SemiJoin/AntiJoin/Project con vincoli della prova e compatibilità fra alternative |
| T QPC e gap | Helper bucket/recognizer/cardinalità | BucketRelation completa e recognizer nominato | Values/calendar input/LeftJoin/Aggregate/Window |
| Record | Invoca lo stesso renderer su slot | Foglia Record con binding risolto, nessuna seconda semantica | Predicato foglia riusato; nessuna espansione dei 26 in nuovi operatori |

Tutte e tre possono rappresentare i SQL provati. Nessuna è sufficiente con soli `certain/possible` indipendenti o con un semplice `ROW_NUMBER` per ogni selettore. La differenza decisiva è quanto costa mantenere e verificare gli obblighi mentre il codice cresce. La tabella è una prova di rappresentabilità di operazioni, non un test di tre compilatori implementati.

## 7. Listing, conteggi ed export senza un risultato persistente obbligatorio

### 7.1 Stessa valutazione, consumi diversi

Quando la valutazione riesce, il compilatore produce logicamente un esito esatto per ogni profilo pertinente, consumato per:

- membership: `truth IS TRUE`;
- conteggi: aggregazioni separate di TRUE/FALSE/NULL sulla relazione a grana profilo;
- listing: ordinamento, limiti e attributi richiesti;
- export: membership, poi consenso/matching/modalità, poi righe di destinazione e trasformazione.

**P:** una fixture con due profili MATCH, uno dei quali senza consenso, e due match remoti per il profilo autorizzato dà segmento=2, profili esportabili=1, righe di consegna=2. Contare il join della destination non misura la popolazione del segmento. Un UNKNOWN escluso dalla membership non diventa un mancato consenso.

Per count e pagina della **stessa richiesta**, un percorso SQL da verificare è un’unica istruzione con relazione di valutazione, aggregato dei totali e relazione della pagina. Non è una garanzia che tutte le forme 1.2 siano eseguibili efficacemente così. `COUNT(*) OVER()` da solo perde il totale se la pagina è vuota: un aggregato sempre presente può fare LEFT JOIN alla pagina, con un indicatore interno che distingue la riga di soli metadati da un profilo. Schema e decoder devono gestirlo esplicitamente. È H, non un protocollo pubblico già approvato né una funzionalità del `Query` attuale.

I conteggi fisici dovranno avere rappresentazione esatta e limiti verificati, con conversione controllata verso l’API scelta. Il `MaxInt32` di `Counts` rimane invariato; non è una giustificazione per troncare un conteggio Unified o per ampliare quel vecchio metodo incidentalmente.

Il nuovo ingresso può mantenere lo scanner corrente per proiezioni con `Column` note, ma servono gestione delle colonne calcolate/nullable e un decoder per il risultato composito. Il solo `Rows` non certifica schema, grana o sicurezza; ne vanno rispettate le convenzioni Scan e controllati gli errori finali.

### 7.2 Coerenza assunta e durata reale

T viene fissato una volta all’inizio della valutazione e reso letterale/parametro comune; le funzioni del dialetto non devono invocare autonomamente “ora” per condizione o batch. Schemi, riferimenti semantici e calendario sono gli input risolti della richiesta. La vista coerente di profili/associazioni/evidenze e il matching valido sono **precondizioni ricevute**, non garanzie che l’attuale codice abbia già implementato.

Per una query/export nel sottoinsieme supportato si può iniziare con una singola istruzione e iterazione sul cursore risultante; questa scelta non impone lo stesso percorso a ogni futura capacità. Il comportamento di buffering del driver, la durata dei dati e la gestione delle risorse vanno verificati; non si dichiara uno streaming a memoria costante per il solo fatto di restituire Rows. Se l’esecuzione fallisce o viene interrotta, il consumo segnala errore: non restituisce un segmento parziale spacciato per completo. Questo non promette rollback delle consegne esterne già avvenute.

Per pagine richieste separatamente, ordinamento totale `(sort keys, profileID)` e seek evitano ambiguità dell’ordine **entro una vista**, ma non congelano membership/attributi fra viste diverse. Un cursore contenente T e fingerprint non crea tale garanzia. La scelta fra rivalutazione dichiarata, vista coerente riutilizzabile fornita dall’infrastruttura o risultato conservato resta separata; il compilatore non impone materializzazione generale o snapshot persistenti.

Anche una transazione non risolve automaticamente quella scelta: Snowflake documenta [READ COMMITTED](https://docs.snowflake.com/en/sql-reference/transactions), con snapshot per statement. Non si estende qui il contratto di lettura fra chiamate, né si riapre il lavoro sulle pubblicazioni. La nuova progettazione non dipende da campi ReadView, tagli di ingestione o lease di generazione obbligatori.

### 7.3 Collegamento all’export già esistente

Riutilizzare la costruzione dei consensi, matching e regole duplicate è utile, ma va separata dalla relazione comportamentale. Oggi `records` combina WHERE Record e consensi e aggiunge il join al matching prima di chiamare `Warehouse.Query`. Per Unified deve consumare una membership già correttamente definita; non inserire consenso o match nei Behavior Scope, nei denominatori o nella base delle selezioni.

Le righe da consegnare possono poi mantenere ordinamento e decoder richiesti da `Records`, incluse le regole `CreateOnly`, `UpdateOnly`, `CreateOrUpdate` e `UpdateOnDuplicates`. Non si può sostituire la sua sorgente Rows con quelle grezze senza adattare lo Scan. Né basta cambiare il campo `Where` del vecchio RowQuery per collegare una CTE generata: `Table` è quotato come un nome, non come SQL.

Riferimenti R: [`records:48`](/Users/marco/src/krenalis/core/internal/datastore/records.go:48), [consensi:61](/Users/marco/src/krenalis/core/internal/datastore/records.go:61), [matching:120](/Users/marco/src/krenalis/core/internal/datastore/records.go:120), [`Records.All:208`](/Users/marco/src/krenalis/core/internal/datastore/records.go:208), [`Store.query:866`](/Users/marco/src/krenalis/core/internal/datastore/store.go:866), [`ProfileRecords:458`](/Users/marco/src/krenalis/core/internal/datastore/store.go:458), [`exportProfiles:64`](/Users/marco/src/krenalis/core/run_export.go:64).

L’ammissibilità Unified resta limitata a Profiles API/Admin ed export di profili verso app/file/database. Le altre pipeline mantengono Record. Non si introduce un secondo sistema pubblico di filtri e non si riusa l’anteprima attuale come certificazione della selezione operativa dell’export.

### 7.4 Contratto completo o errore; percorsi ulteriori non approvati

**D:** timeout, overflow, cancellazione e budget esauriti sono errori tecnici dell’operazione. Non diventano UNKNOWN, NO_MATCH, segmento vuoto o output incompleto presentato come completo. Listing, count, warning e export devono rispettare tutte le informazioni richieste dal loro contratto; non si dichiara successo avendo determinato soltanto una parte della platea o della ripartizione ternaria richiesta.

La prova multiprofilo (§14.4) mostra che un solo profilo costoso può far fallire uno statement che comprende anche un profilo semplice. È un rischio operativo da misurare; non rende necessari né approvati risultati pubblici parziali, stati tecnici pubblici per profilo, export dei soli MATCH già valutati o kernel residui obbligatori in Go. Non si promette rollback di consegne esterne già avvenute: l’operazione fallita deve comunque essere dichiarata tale, non completata con successo.

**H, decisioni ulteriori:** un percorso residuo fuori SQL può essere valutato soltanto con prova di completezza degli input, conservazione dei fatti/owner/compatibilità, stessa vista coerente, esattezza di membership/count/paginazione/export e controllo di memoria, byte, cancellazione e budget. Estrarre alcune righe dalla prima query non dimostra tali proprietà. SQL unico, più istruzioni o residuo locale non sono autorizzazioni a cambiare la semantica né a consegnare risultati parziali. Non viene progettata qui una nuova API.

## 8. Quali parti del rapporto 12 conservare, ridurre o eliminare

Qui “eliminare” significa **non assumere come requisito della nuova architettura**; il file 12 rimane intatto. Un’alternativa tecnica scartata come prerequisito non è vietata per il futuro.

| Parte del 12 | Trattamento raccomandato | Motivazione rispetto a 1.2 e prove |
|---|---|---|
| §§3.1–3.2: pubblico/persistito/semantica separati | Conservare le responsabilità | Non richiedono tre classi o tre pipeline di trasformazione indipendenti |
| §3.2: semantica comune V/E/K/P/T | Conservare | È il confine che evita due interpretazioni nei driver |
| §3.3: pianificatore separato | Ridurre a scelte locali dei moduli | Nessun optimizer autonomo dimostrato necessario |
| §3.4: generatore comune e adattatori | Conservare, con accesso al renderer Record esistente | JSON/tempo/tipi rimangono differenze fisiche; la composizione dell’evidenza è condivisa |
| §4.1: PhysicalRequest/ReadView esteso e retainedResultHandle | Eliminare l’obbligatorietà | T e binding sono necessari; generazioni, ingestion cuts, certificati generali e handle persistenti non sono requisiti normativi 1.2 |
| §4.2: piano fisico autonomo con catalogo di operatori | Eliminare il piano autonomo dalla B consolidata | Algoritmi semantici locali e contratti leggeri non richiedono un secondo albero o lowering |
| §4.3: evidenza conservata prima delle riduzioni | Conservare e precisare | I soli flag certain/possible non dimostrano compatibilità di selezioni, key o part/whole |
| §4.3: massimi non determinabili/copertura | Limitare ai contratti che lo richiedono e ai dati esistenti | Il vuoto registrato è zero; niente eventi esterni ipotetici o completezza esterna generale |
| §4.4: sicurezza e risultati non fidati | Conservare | Validazione dei binding, risorse e valori ricevuti rimane necessaria |
| §§5.1–5.4: protocollo di generazione/pubblicazione e tagli | Togliere dal percorso critico del compilatore | IR/pubblicazione e acquisizione/attribuzione sono lavori autonomi rinviati; la vista valida si assume fornita |
| §5.3: impossibilità di provare assenza con eventi non ancora attribuiti | Correggere nell’uso futuro del 12 | In 1.2 il dominio è la vista interrogabile fornita; la sola latenza non crea UNKNOWN |
| §§5.4, 5.6: conservazione temporanea generalizzata | Eliminare come strategia iniziale obbligatoria | La singola istruzione è un percorso limitato, non prova universale; durata fra richieste e percorsi residui richiedono decisioni distinte |
| §6.1: risultato a grana profilo con verità ternaria | Conservare | È il punto di riuso fra consumatori, non una tabella persistente obbligatoria |
| §6.2: conteggi dei tre esiti e ragioni | Conservare il significato, richiedere solo le proiezioni utilizzate | I conteggi sono per profilo; ragioni sovrapposte non si sommano come popolazione |
| §6.3: ordinamento totale/seek | Conservare come capacità utile, separare dalla durata garantita | Il risultato conservato non è l’unica opzione; T non fissa da solo una vista fra richieste |
| §§6.5, 7.3: facciata Profiles/export comune | Conservare senza imporre un nuovo servizio | Consent/matching fuori dalla membership e stessi moduli per i consumi |
| §7.1: colonne qualificate come adattamento sempre necessario dei renderer | Ridurre | P mostra il riuso tramite proiezioni univoche; qualificazione strutturata resta una possibile ottimizzazione |
| §7.2: persistenza/equality pubblica distinta | Conservare | Nessun cambio a state.Where o ai change detector |
| §8: prove schematiche | Rafforzare con i controesempi di §6 | Ranking nominale e bounds generici non bastano; N2 non autorizza perdita di dipendenze dentro E/P/share |
| §9: protocollo di lettura prima delle verticali | Separare dai lavori del compilatore | Assunzione di input coerente per progettare; garanzie reali restano possibili gate di rilascio |

Non si assume che il 12 sia falso perché più articolato: conserva diversi confini corretti. Il costo ulteriore da giustificare è il piano indipendente e i suoi meccanismi di durata, non la necessità di una semantica comune o di relazioni intermedie durante la generazione.

## 9. Complessità, prestazioni e sicurezza della B ridotta

### 9.1 Astrazioni Go con un’utilità concreta

| Astrazione | Cosa evita | Cosa non dimostra |
|---|---|---|
| Riferimenti risolti a fonte/colonna/owner e literal preparati | Lookup incoerenti, SQL proveniente dai payload, perdita di tipi | Adeguatezza dei dati per ogni profilo |
| Tipi nominali delle relazioni di §5.2 | Contare Refund come ordini, aggregare witness come profili, perdere bucket o owner | Correttezza del SQL interno di ogni modulo |
| Generatore di selezioni congiunte riusabile | Reimplementare selezione in E, pivot/opportunità e passi First/Last P | Un algoritmo generale già pronto per tutte le incertezze |
| Funzioni comuni di chiusura ternaria e quantificazione N2 | Due versioni dei CASE e regole diverse fra dialetti | Una policy universale per tutte le costruzioni |
| Compositore privato di CTE/alias/parametri | Collisioni, riferimenti mancanti, ordine delle dipendenze e duplicazione meccanica | Un optimizer o un SQL AST universale |
| Adattatori dei due dialetti e ingresso eseguibile tipizzato | Duplicazione del significato V/E/K/P/T e del trasporto | Risultati Snowflake reali, buffering o precisione senza prove |

Non aggiungerei `IVCompiler/IECompiler/IKCompiler` per poter sostituire ogni modulo con mock, un secondo modello persistito del piano, un registry delle riscritture fisiche, un interprete universale di scenari o un servizio di materializzazione preventivo. Le interfacce sono giustificate dove cambia davvero il backend; i moduli interni possono essere funzioni e tipi concreti.

### 9.2 Rischi di costo semantici e fisici

- La raggiungibilità di valori di riduzioni additive (§6.1), share e selezioni compatibili può richiedere ricerca di stati; limiti esatti decidono soltanto i casi per cui sono sufficienti. Percorsi e tuple E possono crescere combinatoriamente. N≤10 e P≤5 limitano la forma, non il numero di occurrence candidate. Una traduzione in prodotto cartesiano completo è una prova di espressività, non una strategia di produzione accettabile.
- Spingere filtri di profilo/repertorio/periodo prima delle correlazioni può aiutare **solo** quando conserva la domanda. Non spostare verify nella base E, key prima del selettore P, consenso nei denominatori o finestre corte al posto di FullRelevantHistory.
- Condividere letture identiche è utile, ma va conservato il contesto d’uso. Una condivisione di SQL non elimina i budget normativi degli scope né autorizza a mescolare owner o usare risultati di un altro T.
- Le CTE non promettono né un’unica scansione né materializzazione. PostgreSQL può incorporarle o materializzarle secondo l’uso; la [documentazione WITH](https://www.postgresql.org/docs/18/queries-with.html) descrive anche le differenze fra MATERIALIZED e NOT MATERIALIZED. Misurare piani e cardinalità su entrambi i backend prima di aggiungere hint o strategie locali di esecuzione.
- Calendar a fuso fisso può usare bordi finiti precomputati. Fusi per profilo richiedono operazioni/lookup coerenti senza generare una matrice profilo×tutto il calendario non necessaria. Una storia non delimitata non viene troncata per risparmiare.
- Listing con LIMIT può risparmiare righe restituite, ma un conteggio globale richiede comunque la popolazione pertinente. Non promettere che una pagina da 20 renda economica la valutazione di tutto il segmento.

Questi rischi richiedono algoritmi locali e strategie esatte con budget, non un piano relazionale autonomo. La loro frequenza nei dati reali non è stata misurata: non si afferma che la ricerca combinatoria sia rara o che i costi siano sempre accettabili.

### 9.3 Confini di sicurezza

Gli identificatori provengono da binding risolti del workspace e i valori da preparazione tipizzata; entrambi sono resi con quoting/parametri del dialetto. I frammenti sono prodotti solo da costruttori interni: non c’è un campo SQL configurabile nel filtro. Gli operandi Record già serializzati in modo sicuro possono inizialmente restare literal; non si parametrizza riscrivendo il SQL come testo.

Lo schema risultato deve distinguere JSON null, SQL NULL, numero esatto e bool nullable. Validare valori, conteggi e limiti prima di usarli per allocazioni, cursori e output; riutilizzare gli scanner e `warehouses.Validate*` dove pertinenti, senza assumere che il warehouse sia fidato. Non aggiungere verifiche il cui unico obiettivo sia garantire la verità dei dati del proprietario del warehouse.

Budget di tempo, SQL generato, righe/byte diagnostici e risorse di export vanno definiti. Un limite tecnico superato o una lettura parziale è un errore; non consente tagli nascosti della storia o UNKNOWN inventati. Ragioni diagnostiche limitate e selettive devono preservare le distinzioni normative di 06 §19.7, senza un nuovo unknownPolicy o una API diagnostica pubblica progettata incidentalmente.

### 9.4 Polarità e percorsi rapidi esatti

La valutazione guidata dalla polarità è una **possibile ottimizzazione esatta**, da verificare per ogni consumer. Per l’albero esterno di soli AND/OR di Kleene, l’applicazione `m(MATCH)=true`, `m(NO_MATCH)=m(UNKNOWN)=false` preserva la sola membership: `m(a AND b)=m(a) AND m(b)` e analogamente per OR. Le 18 combinazioni binarie sono verificate in §14.4; l’estensione agli alberi segue per induzione. Il lemma opera su **risultati semantici correttamente chiusi** e non sostituisce la valutazione dei loro producer.

Non autorizza UNKNOWN→NO_MATCH nel filtro, né la perdita di dipendenze interne a E/P/share. Nel controesempio P, sostituire i due flag di tupla UNKNOWN con false perderebbe un MATCH: quelle tuple non sono le foglie semantiche del filtro. None, assenze, AtMost e altre posizioni negative richiedono una prova specifica rispetto alla loro algebra; non si inferisce una regola universale dal lemma positivo. Warning e conteggi ternari restano obbligatori quando richiesti: decidere la membership non equivale a completare quel contratto.

Ogni fast path deve dichiarare e verificare: indipendenza delle scelte o vincoli condivisi preservati; dominio e kind dei valori; guardie; unità, scala e precisione esatta; raggiungibilità degli estremi, distinguendoli da infimo/supremo; richiesta consumata e punto di chiusura. Per somme di contributi determinati e appartenenze opzionali indipendenti, min/max esatti bastano per soglie ordinate senza ulteriori vincoli e per provare che tutti i valori soddisfano un’uguaglianza/intervallo. Non bastano in generale per escludere un valore interno o provare `≠`.

Il lemma di Claude sugli intervalli larghi presume contributi opzionali **indipendentemente selezionabili**, dominio finito e intervallo chiuso: non si applica a tre contributi da 10 governati dal medesimo fatto. In quel caso le somme sono `{0,30}` e `[10,20]`, pur largo quanto ogni contributo, è irraggiungibile. Analogamente, la contiguità dei conteggi di giorni con assegnazioni indipendenti non si trasferisce a fatti correlati: tre eventi che possono avere giorni congiunti `(0,0,0)` oppure `(0,1,2)` danno soltanto 1 o 3 giorni. Sono controesempi analitici verificati localmente, non nuove capacità del linguaggio. La cardinalità esterna QPC continua a usare l’algebra prevista da 06 §14.7, senza inferenza globale fra bucket.

I risultati del 16 su soglie con `minimumWholeWeight` restano rischi per il **ternario completo**; sotto le ipotesi dichiarate, provare MATCH può essere più semplice che distinguere NO_MATCH da UNKNOWN. Ciò non autorizza a omettere tale distinzione quando richiesta. Riutilizzare un algoritmo numerico richiede le sue precondizioni: K-N2, E, V e share non vengono ricondotti a un unico algoritmo di bounds o a un solver generale.

## 10. Prototipi e test effettivamente eseguiti

Questa sezione conserva le esecuzioni della prima stesura del rapporto 14. Non sono state ripetute nel consolidamento; le nuove verifiche mirate sono in §14.4.

### 10.1 Ambiente e artefatti isolati

- Host Go: `go1.27.2 darwin/arm64`; dipendenze vendorizzate, `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`.
- Directory delle prove: `/private/tmp/krenalis-phase4b-14-316yg71w`. Cache e build temporanee sono qui, senza modificare `.gocache` o altri file non tracciati del repository.
- PostgreSQL: immagine locale `postgres:18.6-alpine3.24`, server `PostgreSQL 18.6`, Linux aarch64. Container `krenalis-phase4b-14-316yg71w`, `--network none`, nessun mount host, nessuna porta pubblicata; rimosso al termine. Le query di prova sono SELECT su VALUES/CTE, con sessione read-only; nessun dato applicativo.
- Snowflake: **nessuna esecuzione remota**. Test locali e 28 query di rendering prodotte dall’overlay; parsing/esecuzione SQL e scanner remoti non verificati.

Il primo accesso al socket Docker è stato negato dal sandbox; il successivo accesso autorizzato ha consentito l’esecuzione isolata. Non sono stati usati container o database condivisi.

Gli artefatti temporanei includono `overlay.json`, i due `_projection_probe_test.go`, `record-projection-postgresql.json`, `record-projection-snowflake.json`, `relational_probe.py`, `probe.sql`, `cases.json`, `postgres-output.jsonl` e `prototype-results.json`. Non sono una libreria da integrare o un’implementazione da consolidare. Il rapporto contiene i controesempi, le grane e gli esiti necessari a valutarne il significato anche senza questi file temporanei.

### 10.2 Suite locali esistenti

Comando eseguito, riportato con la sola suddivisione su più righe:

```sh
GOCACHE=/private/tmp/krenalis-phase4b-14-316yg71w/gocache \
GOTMPDIR=/private/tmp/krenalis-phase4b-14-316yg71w \
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off KRENALIS_SKIP_SNOWFLAKE_TESTS=true \
go test -mod=vendor \
  ./warehouses/postgresql ./warehouses/snowflake ./core/internal/filters ./core/internal/state \
  -run '^(TestJSONDecimalRendering|TestJSONExpressionErrors|Test_renderExpr|Test_renderExpr_errors|TestStructuralJSONRendering|TestTypedFilterOrdinarySQLRegression|TestTypedJSONOperandErrors|TestTypedJSONRendering|TestArrayValueRendering|Test_Applies|Test_AppliesWithNestedPath|TestAppliesTypedJSONFixtures|TestAppliesTypedMembershipDecoding|TestAppliesTypedMembershipPresence|TestAppliesValuePredicates|Test_unmarshalWhere|Test_unmarshalWhereJSONDecimals|Test_Where_Equal|Test_MarshalWhere|Test_MarshalWhereValues|Test_WhereCondition_Equal|Test_WhereLogical_UnmarshalJSON|Test_WhereOperator_JSON)$' \
  -count=1
```

**Esito: tutti e quattro i package PASS.** Sono test selezionati di rendering, evaluator Go, operandi e serializzazione/equality. Non sono le suite complete, né nuovi test end-to-end di Unified o una suite differenziale Go/PostgreSQL completa. I test di esecuzione warehouse esistenti esclusi dalla selezione non vengono dichiarati passati.

### 10.3 Overlay Record e query PostgreSQL

```sh
GOCACHE=/private/tmp/krenalis-phase4b-14-316yg71w/gocache \
GOTMPDIR=/private/tmp/krenalis-phase4b-14-316yg71w \
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off KRENALIS_SKIP_SNOWFLAKE_TESTS=true \
KRENALIS_PHASE4B_PROBE_OUTPUT=/private/tmp/krenalis-phase4b-14-316yg71w \
go test -mod=vendor \
  -overlay=/private/tmp/krenalis-phase4b-14-316yg71w/overlay.json \
  ./warehouses/postgresql ./warehouses/snowflake \
  -run '^TestPhase4BProjectionProbe$' -count=1

python3 /private/tmp/krenalis-phase4b-14-316yg71w/relational_probe.py
```

**Overlay: entrambi i package PASS.** L’overlay aggiunge solo un test virtuale per package e chiama `renderExpr`/`serializeValue` originali. Non sostituisce i loro corpi e non crea file nel checkout. Per ogni dialetto genera quattro operatori su sette input: proprietà assente, JSON null, Number 100, String "100", Boolean, Object e Number `9007199254740993.01`.

**SQL: 56 interrogazioni con risultati verificati PASS, più una query di errore tecnico intenzionale.** La firma SHA-256 dello script SQL eseguito è `b2b0b2517b164af495c36a7175e33e8ac3c997f2678d86b44092b78071dc8f48`.

| Gruppo P | Numero di interrogazioni | Cosa è stato verificato |
|---|---:|---|
| Record su proiezione, SQL del renderer reale PG | 28 | TRUE/FALSE/NULL, kind JSON, missing/null, membership e numeri senza float intermedio |
| V | 5 | Count/assenza su quattro profili, raw/NativeCurrency, sum esatta, sum con ignoto, fixture N3 |
| E | 7 | Last assoluto/qualificante, ultima qualificazione incerta, tie con esiti uguali/diversi, contesto condiviso e pairing congiunto |
| K e share | 4 | N2, duplicazione Refund/grana candidata, base incerta, part/whole condiviso |
| P | 8 | Due casi multivalore, lunghezze 2–5, assenza intermedia e ordine incerti |
| T | 3 | QPC con bucket vuoti/ignoti, DST 23/25 ore, max gap ai bordi e ciclo vuoto |
| Listing/count/export | 1 | Segmento=2, autorizzati=1, righe remote=2 |

`SELECT 1/0` ha restituito errore PostgreSQL di divisione per zero, non un valore ternario: il runner lo ha rilevato come fallimento tecnico atteso. Il runner controlla il ritorno di psql e non accetta output parziali come successo.

**Portata precisa:** le query usano relazioni piccole scritte per le fixture; non sono emesse da un compilatore completo del Canonical AST. Le alternative di E e P e parte delle appartenenze incerte sono fornite esplicitamente, anziché derivate automaticamente dai dati. L’esecuzione usa psql, quindi non verifica il futuro trasporto compilato, i suoi bind parameters o gli scanner Krenalis. Nessun risultato misura prestazioni, scala o equivalenza PostgreSQL/Snowflake di Unified.

### 10.4 Estratti riproducibili dei due errori strutturali principali

La chiave comune di P deve essere la stessa in tutti i join del blocco:

```sql
WITH keys(id,k) AS (
  VALUES (1,'x'),(1,'y'),(2,'y'),(2,'z'),(3,'x'),(3,'z')
)
SELECT EXISTS (
  SELECT 1 FROM keys a JOIN keys b USING(k) JOIN keys c USING(k)
  WHERE a.id=1 AND b.id=2 AND c.id=3
) AS path;
-- FALSE; le intersezioni adiacenti controllate separatamente non bastano.
```

In E il contesto che qualifica la base deve sopravvivere al pairing:

```sql
WITH contexts(category) AS (VALUES ('A'),('B')),
events(id,ts,category,score) AS (
  VALUES (1,1,'A',90),(2,2,'B',140),(3,3,'A',100),(4,4,'B',150)
), ranked AS (
  SELECT c.category,e.score,
         ROW_NUMBER() OVER (PARTITION BY c.category ORDER BY e.ts DESC) AS rn
  FROM contexts c JOIN events e ON e.category=c.category
), pairs AS (
  SELECT a.category,a.score AS last_score,b.score AS previous_score
  FROM ranked a JOIN ranked b
    ON a.category=b.category AND a.rn=1 AND b.rn=2
)
SELECT BOOL_AND(last_score-previous_score>=10) AS truth FROM pairs;
-- TRUE; togliere l'uguaglianza del contesto introduce coppie impossibili.
```

Qui i body sono determinati e la base/ordine di ogni alternativa sono noti: BOOL_AND verifica l’esito di questa fixture, non sostituisce la chiusura ternaria generale quando esistono body NULL o selezioni mancanti. La premessa governata di esistenza/univalenza nel dominio A/B non si ricava dal solo tipo Record o da un SQL NULL.

## 11. Limiti ancora non dimostrati e gate di attivazione

### 11.1 Cosa le prove consentono e cosa lasciano aperto

È dimostrato che i due renderer esistenti possono produrre foglie su slot proiettati, e che PostgreSQL esegue i 56 frammenti con gli esiti dichiarati. È motivata la rappresentabilità di questi algoritmi in A/B/C. **Non è dimostrato** un compilatore completo, un insieme minimo definitivo di tipi Go, il supporto di tutte le forme normative, una derivazione generale delle alternative, un trasporto pronto o un vantaggio prestazionale di B.

Non si può approvare come sufficiente una variante che sostituisce la compatibilità interna con bounds indipendenti e restituisce NULL ogni volta che quel modello non decide: potrebbe perdere MATCH o NO_MATCH dovuti alle dipendenze obbligatorie. Se un modulo non è ancora in grado di realizzare una forma, la sua implementazione non è pronta; non si cambia il significato del filtro per nascondere il limite. Analogamente, un costo eccessivo deve emergere come limite tecnico dichiarato, non come un nuovo comportamento UNKNOWN.

### 11.2 Esperimenti minimi, con criteri di esito

| ID | Esperimento richiesto | Criterio per procedere |
|---|---|---|
| X1 — E, punto critico | Proseguire dal sottoinsieme del 15 con oracolo indipendente e fatti condivisi anche nel solo verify | Correggere la lacuna dei due body legati allo stesso booleano; rifiutare esplicitamente i sottocasi non supportati. Conservare selezioni compatibili, guardie, posizioni 10/11 e divieto di fallback; non equiparare concordanza con l’oracolo storico a conformità |
| X2 — K/base/share | Conservare membership distinta dal body, dedup e part/whole; risolvere il confine K-base/body di §14.3 prima dei sottocasi correlati | D e guardie corretti; N2 rimane UNKNOWN sul rimborso univalore. Nessuna interpolazione automatica delle quote discrete V e nessun algoritmo unico per le diverse costruzioni |
| X3 — P | Prima dell’implementazione di P, generare tuple e chiudere fatti condivisi **fra tuple**, oltre a blocco/key, assenze e ordine; oracolo indipendente dai flag del generatore | Export 10:00, ignoto 10:40, Import 11:30, seconda funzionalità diversa da Export entro 60 minuti → MATCH. Conservare stessa key, confini, assenze/estremi, selezioni e ownership; nessun pairwise/greedy o chiusura prematura. Chiarire le parità di §14.3 nei sottocasi che ne dipendono |
| X4 — T | Derivare appartenenze a giorni/bucket e gap con timestamp ignoti, non soltanto passarle come input | Giorni simultaneamente realizzabili corretti; bucket vuoti conservati; DST e calendar/elapsed distinti; N1 invalidità vs estremo ignoto; nessun predecessore inventato per retention |
| X5 — Due database e Record, anticipato | Durante l’implementazione, prima di dipenderne, eseguire sonde Snowflake sulle forme effettive: numeri, NULL, ternari, finestre, subquery, aritmetica esatta; poi fixture comuni attraverso il trasporto | Stesso significato, non necessariamente stesso SQL; 26 operatori/payload, alias quotati, codec, precisione/overflow, NULL/JSON null, collazioni e tempo preservati. Nessun PASS Snowflake dalle sole stringhe o dalla documentazione |
| X6 — Risultati e consumatori | Verificare count/pagina, pagina vuota, output nullable, cancel/errori multiprofilo ed export con consensi/match duplicati | Risultato completo ed esatto oppure errore dell’operazione; totali per profilo e ternari richiesti completi, scanner/Close/Err e risorse corretti. Nessun export parziale approvato implicitamente |
| X7 — Costi delle strategie locali | Misurare nella B ridotta fast path e algoritmi esatti, con input runtime e casi multiprofilo, incluse somme additive e share | Precondizioni verificate, risorse misurate, budget→errore tecnico; nessuna affermazione di rarità nei dati reali senza misure. Non occorrono tre compilatori o un piano autonomo |

Gli oracoli finiti servono a verificare i moduli; non propongo di esportarli come nuovo algoritmo generico del prodotto. Occorre esplicitare l’insieme di fatti condivisi e di compatibilità gestito da ogni firma. Un contratto `SelectedTupleRelation` è utile soltanto se il suo costruttore mantiene davvero tali garanzie.

X1–X4 sono gate semantici dei rispettivi moduli; i risultati del 15 ne coprono soltanto sottoinsiemi, con la correzione E ora registrata. Le ambiguità non risolte non bloccano automaticamente le funzionalità indipendenti. X5 è necessaria per certificare i due backend; X6 per integrare consumatori; X7 per scegliere strategie ed eventuali strutture ulteriori. Non devono diventare un pretesto per anticipare tutti i dettagli del wire format o i lavori IR/writer/matching, che restano autonomi. Le prove infrastrutturali sulla vista coerente rimangono possibili prerequisiti di rilascio delle funzionalità che le richiedono.

Il confronto minimo A/B del 15 sostiene una sola implementazione con controlli ai confini. X7 misura ora strategie locali dentro la B scelta; non riapre il confronto generale A/B/C. Un algoritmo non ancora conforme rimane non attivabile, non viene mascherato da UNKNOWN.

## 12. Prima verticale e sequenza successiva al consolidamento

**D:** la prima verticale è **V con count, esistenza e assenza sugli eventi registrati**, risultati ternari e riuso dei renderer Record. Comprende vuoto registrato, predicati NULL decisivi, distinzione JSON raw/NativeCurrency, grana e guardie pertinenti. Le capacità non conformi restano non attivabili: UNKNOWN non sostituisce una funzionalità mancante.

Ordine successivo, non implementazione avviata con questo incarico:

1. Preservare `core.Filter`, `FilterCondition`, preparazione, 26 operatori, `state.Where`, serializzazione ed equality; definire il minimo accesso interno ai renderer e il trasporto tipizzato.
2. Realizzare la verticale V circoscritta e i controlli multiprofilo con la vista coerente assunta; anticipare le sonde Snowflake X5 sulle forme effettivamente introdotte. Il renderer conserva TRUE/FALSE/NULL fino al consumo corretto.
3. Verificare listing/count ed errori dell’operazione con X6, senza ampliare la verticale all’intero kernel numerico, a tutte le share o a un residuo universale.
4. Estendere E dopo X1 e il rifiuto/supporto esplicito del caso verify condiviso. Sbloccare K per i sottocasi con confine base/body definito e X2 superato; lasciare in sospeso gli altri (§14.3).
5. Implementare P soltanto dopo X3 esteso ai fatti fra tuple. Aggiungere T con X4 e i vincoli N1, preservando l’algebra QPC e la distinzione fra ordine semantico e tecnico.
6. Valutare separatamente riduzioni additive e share avanzate con algoritmi specializzati esatti, guardie, precisione e costi X7. Nessun obbligo preventivo di kernel fuori SQL.
7. Collegare gli export app/file/database alla membership già definita, applicando dopo consensi, matching e modalità; verificare risorse, errori e completezza prima dell’attivazione.

**Ancora aperti:** nomi/firme e raggruppamento dei contratti interni, trasporto/scanner, forma pubblica/persistita delle nuove `FilterRule`, durata dei risultati tra richieste, strategie esatte dei moduli e chiarimenti locali di §14.3. Non rimane da scegliere fra A/B/C: B ridotta è l’architettura di riferimento confermata. L’eventuale calcolo residuo, i risultati parziali e nuovi stati tecnici pubblici richiedono decisioni ulteriori; non sono prerequisiti approvati.

Nessuna scelta del rapporto 12 è adottata automaticamente. **Identity Resolution/pubblicazione Profiles, acquisizione asincrona/attribuzione KPID e destination matching rimangono rinviati e non risolti.** Non bloccano la progettazione attuale; possono condizionare il rilascio e non si dichiara che la vista coerente o il matching valido siano già garantiti dall’implementazione.

## 13. Verifica documentale e perimetro del consolidamento

La baseline verificata anche per il consolidamento è `typed-filter-values@c23777219db49fbf6164b88b4e74ca55a5830f64`, senza modifiche applicative né staging preesistente. I rapporti 14, 15 e 16 erano non tracciati; erano presenti anche file estranei, inclusi 11/12 e i due input di Claude. Questi ultimi restano esclusi dal commit dedicato richiesto, come gli altri file non pertinenti.

È modificato soltanto il contenuto del 14. Il 15 conserva SHA-256 `6c9e861158676eb68b49e2aa4c67619f9e3742ec8fa67bc539c1180ae05be3b2`; il 16 conserva `0889f13ccfc9ee1be3bab3bf83a468175d4d017d4bc36b4cb1a6b97ecfa1616a`. La firma del 14 prima di questo intervento è `c7ec8a5451fa45d1c3b469cfbef77dc2c989c6169d41c7ed7b17fac349ec5dc7`. Il commit include esclusivamente questi tre rapporti; non cambia lo stato normativo della 1.2 né approva il rapporto 12.

I controlli del consolidamento comprendono SHA-256 di tutti gli altri file preesistenti in `filtering`, identità byte per byte dell’08 con la sezione finale del 07 v1.2, collegamenti locali, riferimenti di sezione, blocchi Markdown, staging esatto e `git diff --check`. La conservazione integrale di 06/07/08 preserva anche i 26 operatori, il contratto `core.Filter`, costruttori, limiti, fixture JSON raw/NativeCurrency e decisioni N1/N2/N3. Le prove runtime nuove sono solo quelle di §14.4; §10 e i rapporti 15/16 restano evidenze storiche.

I link ai rapporti 11/12 sono verificabili nel checkout attuale ma i loro file restano non tracciati ed esclusi da questo commit. L’appendice di §14.5 rende riproducibili le nuove prove senza dover consolidare il kit esterno di Claude. Nessun codice applicativo, driver, benchmark o specifica approvata è modificato. Nessun motore di produzione della Fase 4B è avviato.

## 14. Emendamenti dopo la revisione indipendente di Claude

### 14.1 Decisione, fonti e ciò che non viene adottato

Consolidamento del 27 ottobre 2026 sulla baseline indicata in §13. Letto il testo integrale `filtering/revisione-claude.txt` (SHA-256 `f6f3c885d40fd7b90f58ebd56d725c8c47a0da822bc7a02bfe9bec3501209ee4`), esaminati script, controesempi e risultati del kit `filtering/krenalis-4b-prove-riprodotte.zip` (SHA-256 `2c7af482962a846824a76e33659f3bf24cb17620385a204505578a02494ba0e4`). Il kit è stato estratto soltanto in una directory temporanea, dopo controllo di dimensioni e path; i suoi risultati PostgreSQL 16.15 sono evidenza della revisione, non nuove esecuzioni di questa sessione.

| Punto della revisione | Trattamento nel consolidamento |
|---|---|
| B ridotta, senza IR/optimizer/lowering | **D:** confermata dal titolare come architettura di riferimento (§1) |
| B2, chiusura P e somme additive | Correzioni recepite in §§5.3, 6.1, 6.5 e X3; casi riprodotti (§14.4) |
| C3, E con fatto condiviso soltanto nel verify | Difetto riprodotto; rifiuto sperimentale esplicito in copia isolata; gate X1 prima del supporto (§6.2) |
| D1, polarità; D2, percorsi rapidi | Ottimizzazioni esatte possibili con ipotesi e confine di consumo dichiarati (§9.4); nessuna perdita di ternario richiesto |
| B1, costo e timeout multiprofilo | Rischio confermato; **D:** completo/esatto oppure errore tecnico. Non vengono adottati gli stati pubblici per profilo, l’incompletezza pubblica o un residuo Go obbligatorio suggeriti da Claude (§7.4) |
| C2, interpretazioni normative | Esame nei testi completi 06/07, nell’estratto 08 e nella decisione N2 del 13; esiti e limiti in §14.3. Nessuna modifica normativa implicita |
| Tre o sei contratti | Nessun numero obbligatorio; conservare tutte le informazioni necessarie e raggrupparle dove semplifica (§5.2) |
| C1/C5/C6/C8, portabilità | Anticipare sonde Snowflake reali durante l’implementazione sulle forme usate. Semantica uguale non significa SQL identico; non adottare automaticamente proposte su precisione di T o calendario senza verifica |
| C9, diagnostica più costosa della membership | Warning e conteggi ternari richiesti rimangono parte del risultato completo; nessuno stato pubblico di ripartizione “non valutata” è introdotto |
| Avvisi al salvataggio, politiche export parziali, scheduling residui | Opzioni ulteriori di prodotto/esecuzione da discutere; non requisiti approvati né nuove API |

La nuova architettura non richiede la conformità retroattiva di tutti gli esperimenti: i rapporti 15/16 restano evidenze storiche con il proprio perimetro. La concordanza tra compilatore e oracolo che condividono una stessa omissione non dimostra conformità; il caso E lo mostra concretamente. Non viene dichiarato un PASS complessivo della 1.2 o di Snowflake.

### 14.2 Confine della chiusura e delle inferenze

| Costruzione | Informazione da conservare | Punto di chiusura |
|---|---|---|
| V additiva / share | Stessa popolazione, pesi, guardie e fatti comuni agli operandi; valori effettivamente compatibili | Test terminale della singola costruzione, senza esportare un valore/una formula nuova |
| E | Base, ordine/selezione e fatti condivisi letti nella qualificazione **o nel verify**, anche fra occurrence diverse | Test sulla selezione congiuntamente compatibile, poi risultato della E |
| P | Tupla completa, eventuale key comune, estremi degli scope negativi e fatti che governano **più tuple alternative** | Esistenza complessiva del medesimo percorso per profilo, poi risultato della P |
| K | Base/identity e body distinti; vincoli interni delle singole costruzioni | Body individuale, poi quantificatore N2 secondo 06 §12.6. Nessuna inferenza globale fra body di candidati distinti; confine base/body misto da chiarire sotto |
| T/QPC | Dipendenze interne al recognizer del bucket, serie completa e risultati individuali | Recognizer, poi cardinalità QPC secondo 06 §14.7; nessuna inferenza simbolica obbligatoria fra bucket |
| AND/OR esterno | Risultati semantici già chiusi | Logica forte di Kleene di 06 §20; nessun ragionamento globale sui fatti delle rule sorelle |

Nel controesempio P, `per alternativa → esiste una tupla → chiusura del percorso` conserva la dipendenza; `chiusura ternaria per tupla → OR` la perde. È una descrizione del confine semantico, non l’obbligo di enumerare mondi in produzione. Il caso usa predicati governati sullo stesso valore presente ma ignoto: non permette di applicare il terzo escluso ai NULL dei confronti Record raw.

### 14.3 Verifica dei quattro punti C2 contro la normativa approvata

I riferimenti sono alle versioni 1.2 presenti nel commit di baseline; le linee permettono di verificare il testo completo, oltre al riepilogo. Non viene proposta una prevalenza automatica del prototipo sul documento normativo.

**C2-K — Base correlata e body N2.** Riferimenti: 06 §§12.1–12.3, §12.6 (righe 577–589), §19.4 (1049–1065); 07 §9.1 (965–1005), §9.6 e F3-K; 08 F3-K; rapporto 13 §M.2 e §N. N2 è esplicita su **base determinata** e risultati individuali; non determina una base incerta né cancella deduplicazione e vincoli interni. Casi minimi:

- Due ordini certi, un Refund con orderId ignoto univalore, body “nessun rimborso”: i due body sono UNKNOWN e `Some` è **UNKNOWN**, come 06 §28.24. È riprodotto dal caso K01; non si completa la chiave del Refund globalmente per rafforzare il quantificatore.
- O1 certo, O2 opzionale, entrambi qualificanti quando presenti: quota 1 in ogni base, minimo 1 già soddisfatto → **MATCH** per share=1 (K02). Se O2 è non qualificante, share=1 è **UNKNOWN** (K03); con soli body determinati le quote `{1,1/2}` escludono 3/4 → **NO_MATCH** (K04), in applicazione del test sui valori compatibili di 06 §19.4. Non è l’inviluppo dei body UNKNOWN di N2.
- Una occurrence con identity O1 e un’altra con identity governata univalore in `{O1,O2}`, tutti i body veri: la base deduplicata ha 1 oppure 2 candidati, share=1 → **MATCH** (K16). La variazione di D non implica variazione della quota; non si contano due record come due identity certe.

**Limite non risolto K:** i testi non specificano completamente come condizionare i body quando lo **stesso fatto comune** governa anche la presenza di candidati diversi. Modello minimo a livello delle evidenze: O1 certo, O2 presente se `g=false`; body di O1 legato a `g`, body di O2 certamente vero. Chiudere il body O1 prima dei contesti di base dà UNKNOWN, e `Some` può restare UNKNOWN nella base senza O2. Condizionarlo alla base può invece provare Some in entrambe le alternative. Serve chiarire il confine fra conservazione della base e inferenza fra candidati vietata da N2, con una fixture autorizzata e il suo binding. Il prototipo non decide questa questione normativa: il sottocaso misto resta **non attivabile**; non viene scelto uno dei due esiti come conforme. Non è stata individuata una coppia di prescrizioni che imponga contemporaneamente entrambi: è una lacuna di specificazione locale, non una modifica già approvata di N2. I casi con base determinata e gli esempi precedenti non ne dipendono.

**C2-NULL — Fatto governato e NULL Record.** Riferimenti: 06 §§8.2, 19.1–19.4 (1011–1065), §§28.19–28.20; 07 §§3–3.1 (167–225), §4.1.2 (318–351), §14.3 (1754–1772), §16.3; 08 F2 “Predicate chiusi”, F6 e F7. Un riferimento governato e i suoi vincoli possono individuare un unico fatto comune, come `g` in E. **Il solo NULL SQL non certifica né identità del fatto né un dominio di completamenti.** Missing raw, kind non applicabile, valore business ignoto, deadline e ordine incerti possono tutti produrre NULL, con cause e obblighi diversi.

Casi minimi ed esiti:

- JSON `{}`: `currency is "EUR"` e `currency is not "EUR"` valgono entrambi NULL; il loro OR Record è **UNKNOWN**, non una tautologia MATCH. Verificato con SQL PostgreSQL; non è una nuova chiamata al renderer reale (§10 ne conserva le prove storiche).
- JSON `{"currency":null}`: confronto raw con String EUR **FALSE**; assenza di Purchase raw-EUR **MATCH**. La stessa proprietà letta come NativeCurrency sconosciuta dà confronto e assenza **UNKNOWN**, secondo la fixture normativa §28.20, conservata integralmente.
- Due letture dello stesso booleano governato ignoto nel verify E possono avere i conteggi 0/2 di §6.2: questo vincolo dichiarato non si ricava dalla semplice presenza di due NULL restituiti dal renderer.

Il riuso Record conserva gli operatori e la loro logica SQL, con path/payload/preparazione invariati; non richiede nuovi registry. L’attribuzione di dipendenze semantiche a combinazioni raw/governate non dimostrate rimane non attivabile per quei sottocasi, senza disabilitare i normali Record o i count/assenze evidence-aware già definiti. Non si introduce un modello generale che “risolva” ogni NULL in un valore reale latente; sarebbe in contrasto con i casi raw sopra e con il divieto di semplificare `x=x` su valori ignoti.

**C2-share — Limite numerico e insieme discreto.** Riferimenti: 06 §§10.1, 10.3–10.5, §12.6 (581–585), §19.4 (1058–1065); 07 §§4.1.1, 7.3 (777–799), 9.1 (999–1005), 9.6; 08 F3-V e F3-K. Per V, “la quota è compresa fra S/D e (S+U)/D” è un limite; la regola conclusiva sui valori compatibili non rende raggiungibile ogni punto del suo inviluppo reale. Con una occurrence part certa e una part possibile, D=2, il test `V.FrequencyShare=3/4` è **NO_MATCH**, dato `{1/2,1}`. La stessa discretezza interessa gli intervalli che cadono interamente nel vuoto fra quote raggiungibili.

Su **K con base determinata**, invece, la specifica prescrive espressamente di usare l’inviluppo composizionale per Share, come per Cardinality; il caso K18, D=2/S=1/U=1, `K.Share=3/4` resta **UNKNOWN** secondo quella regola. Questo trattamento deriva dalla disposizione specifica per K Share di 06 §12.6 e 07 §9.1, non dal solo nome “N2”, e non viene generalizzato alle quote V o alle alternative di base con body determinati. I test SQL mostrano entrambi gli esiti, senza uniformarli. Non emerge una contraddizione necessaria fra la frase “è compresa” e la regola finale: l’uso indiscriminato dell’intervallo come insieme esatto dei valori V sarebbe l’interpretazione errata.

**C2-ordine — Parità e simultaneità.** Riferimenti: 06 §11.2 (479–483), §13.2 (608–616), §§13.3–13.7, §17.3; 07 §8.4 (953–959), §12.4 (1558–1564), §14.4; 08 F2 “Scope e tempo”, F3-E/P. Timestamp distinti provano precedenza; a timestamp uguali è necessario ordine ulteriore affidabile del dominio. Nessun ID, ordine di arrivo o `ROW_NUMBER ORDER BY id` può crearlo. Se il contratto certifica a≺b, `a THEN b` può essere **MATCH con ritardo zero**; se certifica b≺a, quel percorso direzionale è **NO_MATCH**. Con orientamento non determinato e requisiti che cambiano fra gli ordini ammessi, l’esito è **UNKNOWN**, come per le selezioni E con body diversi; se tutte le selezioni compatibili hanno body vero, E può essere MATCH senza scegliere arbitrariamente il testimone.

**Limite non risolto P:** due occurrence dello stesso scope, stesso timestamp, nessun ordine ulteriore; P chiede due occurrence distinte di quello scope in sequenza. Se il contratto garantisce un ordine totale pur senza renderne nota l’orientazione, entrambe le estensioni lineari contengono una coppia valida. Se è ammessa anche vera simultaneità senza precedenza, esiste un’alternativa senza percorso. I testi non definiscono completamente l’insieme di completamenti per questa situazione. La sonda SQL di §14.4 mostra **MATCH nella sola ipotesi totale** e un’alternativa falsa nella simultaneità; non dichiara uno dei modelli conforme per default. Il sottocaso resta **non attivabile** finché contratto dell’ordine e fixture normativa non lo risolvono. Non si usa UNKNOWN per nascondere questa mancanza di definizione, né un tie-break per fingere una prova. I percorsi con ordine affidabile non sono bloccati da tale limite.

### 14.4 Verifiche mirate effettuate in questa esecuzione

Artefatti isolati in `/private/tmp/krenalis-consolidate14-4p3_a94m`. Python 3.13.2, macOS 27.0.1 arm64; server PostgreSQL **18.6** aarch64 nell’immagine locale `postgres:18.6-alpine3.24`, ID `sha256:d7a8005067f55a2f8ea369e767545fd467b1943a4f0f1a48b7dc44f06c366109`. Container nuovo con `--rm --pull=never --network none --memory 512m --cpus 2`, nessuna porta o mount, rimosso al termine. `jit=off`, `work_mem=4MB`, timeout ordinario 5 secondi. Nessuna lettura/scrittura su warehouse applicativi; nessuna esecuzione Snowflake.

| Verifica nuova | Risultato effettivo | Portata |
|---|---|---|
| Kit Claude `cx/p_certain_tuple_vs_exact.sql` | Chiusura per tupla NULL; chiusura su esistenza per alternativa TRUE | Conferma il difetto della regola incompleta P; non un generatore P completo |
| `cx/v_sum_subset.sql` | Somme e quattro esiti della tabella §6.1 confermati | Otto scelte di inclusione, sette somme distinte; nessuna misura di scala |
| `cx/share_envelope_vs_exact.sql` | Inviluppo→NULL; quote compatibili→FALSE | Due occurrence V; lettura normativa distinta da K |
| E originale del 15, flag condiviso solo nel verify | Modello accettato; SQL NULL; oracolo NULL con un contesto | **Difetto riprodotto**, non PASS di conformità |
| Oracolo diretto e SQL dei due valori del fatto governato E | Conteggi `[0,2]`; esito FALSE in entrambe le alternative | Enumerazione indipendente di quel singolo fatto; nessun solver generale |
| `guarded15.py`, esperimento di ammissione | Entrambi gli ingressi, compilatore e oracolo, rifiutano il caso con `PrototypeUnsupported` | Rifiuto dell’implementazione sperimentale, non invalidità del filtro 1.2; nessun risultato ternario prodotto |
| K01/K02/K03/K04/K16/K18 del 15 | Rispettivamente NULL/TRUE/NULL/FALSE/TRUE/NULL | Conferma i risultati del micro-modello; non risolve il caso misto base/body di §14.3 |
| SQL raw missing/JSON null | Equal/NotEqual mancanti NULL; OR NULL; Equal su JSON null FALSE | Sonda di composizione, non suite differenziale completa dei renderer |
| P, tre ipotesi d’ordine su due eventi pari | Coppia presente nei due ordini totali, assente nell’ipotesi simultanea | Esperimento discriminante **non qualificato come conformità** |
| DP runtime del 16, W03 da solo, budget 100 ms | FALSE, 4 stati finali/7 cumulati | 39 ms circa complessivi inclusi Docker/psql; non tempo server |
| Stesso DP, W03 e power20 in uno statement; power20 da solo | Entrambi errore SQLSTATE **57014**, budget 100 ms | 138/142 ms circa inclusi trasporto; nessun risultato semantico accettato per le operazioni fallite |
| Test contrattuali storici del prototipo 15 | **7 test PASS** | Dimostrano che quei test non intercettavano il nuovo difetto E |
| Guard isolato sulle fixture E storiche | SQL identico in **739** casi; stesso oracolo in **738** casi esaustivi | Un dominio int64 resta non enumerabile come già dichiarato dal 15; non sono 739 nuove esecuzioni SQL |
| Kleene e precondizioni dei fast path | **18** combinazioni binarie corrette; controesempi correlati per intervallo largo e giorni distinti verificati | Prove locali e argomento induttivo; nessuna frequenza stimata sui dati reali |

Totale: **16 esecuzioni SQL completate**, comprese versione server e caricamento della sorgente temporanea; **2 timeout intenzionali** verificati separatamente. Le 14 query di contenuto completate includono la riproduzione del difetto E e la discriminazione fra ipotesi d’ordine: non sono 14 PASS normativi. Il timeout multiprofilo usa 24 righe lette da `experiment_events` nel container; il generatore è quello originale del 16 e non incorpora i pesi nel testo della query. La prova di Claude usava un secondo di budget su PostgreSQL 16.15: non ne vengono attribuiti qui tempi o versione.

L’esperimento correttivo è un wrapper su copie verificate dei sorgenti storici, non una riscrittura incorporata nel 15: `check_shared_flags` controlla i predicati di qualificazione **e** body prima di chiamare compilatore/oracolo. Nel 15 originale, `compiler.py:161–166` limita il controllo a `e.qualifier`; `oracle.py:85–86` attiva i domini dei flag soltanto in base al qualifier; il ramo Latest usa poi la quantificazione dei body senza quella correlazione. Non viene dichiarato che il wrapper risolva altri difetti o implementi tutte le dipendenze E.

Comandi delle verifiche nuove:

```sh
python3 -I -B /private/tmp/krenalis-consolidate14-4p3_a94m/verify14.py
```

Il programma esegue i sette test con `unittest`, le verifiche locali e le query via `docker exec ... psql -XAtq -v ON_ERROR_STOP=1`. Con un exit non-zero scarta il risultato dell’operazione; verifica SQLSTATE 57014 nei due casi deliberati e rimuove il container in `finally`. Il primo tentativo locale con `python -I` non aveva il path dei moduli, e un tentativo preliminare invocava l’oracolo anche sul caso int64 già dichiarato non esaustivo: entrambi sono stati corretti nel solo runner prima della prova completa, senza cambiare compilatore, oracolo o attese storiche. I risultati sopra appartengono all’esecuzione completa corretta.

Non sono state rieseguite le suite applicative Go, le 56 query storiche del 14, tutte le 1.022 query del 15 o gli esperimenti di scala del 16. Non occorrono a questo intervento documentale mirato e non vengono dichiarati nuovamente passati. Gli input e gli output della revisione restano distinti dalle nostre verifiche.

### 14.5 Riproduzione delle correzioni isolate

L’archivio seguente conserva il runner, il guard, le tre query originali del kit usate qui e gli output mirati. Non contiene codice di produzione né modifica gli archivi storici. I sorgenti di `source15/` e `source16/` si estraggono dagli archivi incorporati nei rispettivi rapporti, verificando le firme riportate sotto. La riproduzione non richiede l’inclusione dei due file di Claude nel commit.

SHA-256 dell’archivio mirato: `1b814679ade3cdd0606a839158eb7d560e8c71f962c8b3cf9345ad8ce96010b9` (11399 byte). Sorgenti storici richiesti: archivio 15 `fbe836dfd772f4a2aea875eb2c7a26bae90582abec161ae8abf7c667577df144`; archivio 16 `e4ec2e5b896ba1861c1d94f07e803986212341b6cec7cb6dd18c08bb9a0552c3`.

Estrazione verificata in una nuova directory temporanea (sostituire il solo percorso iniziale):

```sh
python3 -I -B - /percorso/krenalis/filtering <<'PYEXTRACT'
from pathlib import Path, PurePosixPath
import base64, hashlib, io, re, sys, tempfile, zipfile
sources = Path(sys.argv[1])
target = Path(tempfile.mkdtemp(prefix="krenalis-14-review-"))
archives = [
    (14, "CONSOLIDATION14", "1b814679ade3cdd0606a839158eb7d560e8c71f962c8b3cf9345ad8ce96010b9", target),
    (15, "PROTOTYPE", "fbe836dfd772f4a2aea875eb2c7a26bae90582abec161ae8abf7c667577df144", target / "source15"),
    (16, "COMPLEXITY16", "e4ec2e5b896ba1861c1d94f07e803986212341b6cec7cb6dd18c08bb9a0552c3", target / "source16"),
]
for number, marker, expected, output in archives:
    document = next(sources.glob(f"{number}-*.md")).read_text()
    start = document.index("<!-- BEGIN " + marker + " ZIP -->")
    encoded = re.search(r"```base64\n(.*?)```", document[start:], re.S).group(1)
    data = base64.b64decode(encoded)
    assert hashlib.sha256(data).hexdigest() == expected
    with zipfile.ZipFile(io.BytesIO(data)) as archive:
        assert sum(item.file_size for item in archive.infolist()) < 2_000_000
        for item in archive.infolist():
            path = PurePosixPath(item.filename)
            assert not path.is_absolute() and ".." not in path.parts
            assert (item.external_attr >> 16) & 0o170000 != 0o120000
            if item.is_dir():
                continue
            destination = output / path
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(archive.read(item))
print(target)
PYEXTRACT
```

Dopo aver ispezionato gli script, eseguire `python3 -I -B <directory-stampata>/verify14.py`. Richiede Docker e l’immagine PostgreSQL indicata già presente; non scarica immagini o dipendenze. Gli output `evidence/` verranno rigenerati nel solo ambiente isolato. I timeout sono misure dipendenti dall’ambiente: un budget non scattato va riportato come tale, non forzato in un risultato semantico.

<!-- BEGIN CONSOLIDATION14 ZIP -->
```base64
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
```
<!-- END CONSOLIDATION14 ZIP -->
