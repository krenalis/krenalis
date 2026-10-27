# Revisione semantica mirata — Rapporto definitivo 1.2 a mondo chiuso

**Stato: rapporto definitivo della revisione 1.2 approvata.** L’approvazione esplicita del titolare del progetto rende i documenti 06/07/08 v1.2 la nuova baseline normativa e tecnica, incluse N1, N2 e N3. È confermata espressamente N2: quantificazione composizionale dei risultati ternari individuali K, senza inferenza globale fra candidati distinti e preservando le dipendenze interne obbligatorie. Le sezioni K/L/M conservano la storia delle verifiche; il consolidamento è registrato in §N.

## A. Baseline iniziale, fonti e ambito effettivo

- Checkout di partenza verificato: `typed-filter-values@25f639d84eb35815af5d1062c86545b65b97d720`.
- Fonti principali: documenti approvati 06, 07 e 08 v1.1; benchmark storico 03; rapporti informativi 11 e 12. Il rapporto 12 è una proposta architetturale non approvata e resta intatto.
- Le stesure precedenti hanno creato soltanto [06 v1.2](06-specifica-semantica-riconciliata-v1.2.md), [07 v1.2](07-ast-canonico-riconciliato-v1.2.md), [08 v1.2](08-ast-canonico-estratto-v1.2.md) e questo rapporto, senza commit o modifiche applicative. Il consolidamento definitivo esclusivamente documentale è descritto in §N.
- Analisi documentale; nessuna esecuzione di filtri, test Go, PostgreSQL o Snowflake. I risultati nelle fixture sono requisiti attesi, non esiti di implementazione.
- Il benchmark 03 è memorizzato in UTF-16BE senza BOM: è stato decodificato soltanto in una copia temporanea per l’analisi. I byte del file originale sono preservati.

## B. Decisione recepita e variazione minima

La versione 1.2 usa un solo universo event-based: **occurrence registrate, disponibili e attribuite al profilo nella vista coerente della query**. Non si aggiungono possibili occurrence perché potrebbero esistere nei sistemi esterni. Zero eventi pertinenti è normalmente un insieme vuoto e count zero.

La verità è relativa a quel patrimonio: “mai” significa mai nella storia interrogabile; “primo” è il primo registrato pertinente. Retention/cancellazione possono cambiare questi risultati e devono essere visibili come limiti del dato. Non attestano una vita del cliente priva di eventi esterni o eliminati.

**Variazione AST minima:** nessuna nuova primitiva, variante, campo o payload. Rimangono `ObservationPeriod.FullRelevantHistory(RelationshipHistoryRef)` e `AvailableHistory(DeclaredRetentionHistoryRef)`; cambia il significato del primo e si precisa il perimetro del secondo. AvailableHistory è un intervallo esplicito; FullRelevantHistory è tutta la storia interrogabile. Possono coincidere nei dati senza diventare alias canonici universali. Sono preservati costruttori, firme dei body, ownership, operatori, limiti strutturali ed esclusioni. La successiva revisione di Pro aggiunge un vincolo di ammissibilità per MaximumGap e rende esplicita la scelta semantica composizionale di K e la rilevanza degli ignoti nei test di prevalenza (§M), senza cambiare le dichiarazioni AST. Il solo cambio nel codice illustrativo del 07 è il binding H verso `RegisteredQueryableHistory@1`, non una nuova dichiarazione ADT o una reinterpretazione del precedente ID immutabile.

La risoluzione di `RelationshipHistoryRef` usa le definizioni governate pertinenti della fonte/repertorio già previste dal 07. L’uso 1.2 richiede il significato dichiarato di storia registrata interrogabile; una definizione 1.1 della relazione reale deve avere una nuova identità o `meaningRevision` immutabile, risolta e adottata esplicitamente. Il cambio del tag/label o un alias non sono sufficienti. Non si aggiungono costruttori, `DefinitionKind` o registry alle condizioni Record; il solo nome illustrativo H non sostituisce questi controlli.

Questo è un cambiamento intenzionale della semantica Unified rispetto alla 1.1, non una modalità configurabile: nessun `unknownPolicy`, `legacyMode` o selettore di mondo aperto/chiuso. Non cambia la semantica Record implementata da `d69e6168`. La versione 1.2 è approvata; l’adozione di eventuali definizioni governate nuove resta esplicita; non si decide qui la migrazione di ipotetici filtri Unified precedenti.

## C. Inventario delle prescrizioni interessate

| Documento / sezioni | Modifica o verifica |
|---|---|
| 06 §§1–3, nuovi §§3.1–3.2 | Stato normativo approvato, universo registrato, vuoto valido, retention e assunzioni/rinvii infrastrutturali |
| 06 §8.2, ultima prescrizione fonte/schema/predicato | Mantiene N4 e indipendenza dall’executor; la mancanza di eventi non si confonde con la perdita di valori decisivi nei record o con garanzie contrattuali mancanti |
| 06 §§9.1 e 9.4 | Behavior Scope e periodi storici senza attestazione dell’inizio reale; unioni di repertori senza intersezioni di copertura inventate |
| 06 §§10.1 e 10.4 | Vuoto/assenza, distribuzioni e novità riferiti al registrato; valori ignoti esistenti conservati |
| 06 §§11.2–11.3 | Primo/ultimo/n-esimo e latest N; insufficienza certa della base distinta da posizione incerta per record esistenti |
| 06 §§12.4 e 12.6 | Prime consegne registrate prima dei filtri; candidati event-based distinti da inventari autorevoli incompleti |
| 06 §§13.1–13.2 e 13.5 | Assenze/adiacenza nei percorsi e deadline; T>d invariato, rimossa la certificazione esterna generale della finestra |
| 06 §§14.2, 14.4–14.5 e 14.7 | Bucket vuoti, gap e ritorni sul registrato; predecessore reale registrato obbligatorio; incertezza dei dati esistenti preservata |
| 06 §15.4 | Precisazione di confine; inventari, lifecycle e garanzie dei named scopes non vengono certificati dal mondo chiuso |
| 06 §§19.1–19.7 e 25.3 | Cause di UNKNOWN, limiti quantitativi, osservazione e spiegazioni distinti dalla pretesa di completezza esterna |
| 06 §§20.1, 22.1, 22.3 e 28.1; 07 §§5.1–5.3, 7.5, 19 e modello finale F2 | Correzioni mirate successive: versione normativa corrente, tempo/copertura per dominio, premesse degli acceptance e risoluzione delle definizioni governate della storia; dettaglio in §L |
| 06 §§26–27 | Catalogo e deferred preservati; nessun recupero implicito della relazione reale dalla storia conservata |
| 06 §§28.1, 28.3, 28.9, 28.12, 28.14–28.15 | Premesse delle fixture aggiornate; UNKNOWN ancorati a dati esistenti, non a ipotetici eventi mancanti |
| 06 nuovi §§28.21–28.22 | CW01–CW21 e confine rispetto al benchmark storico/out-of-sample |
| 06 §§29.3–30 e chiusura | Approvazione della revisione, adozione esplicita delle definizioni governate e versionamento dei significati |
| 07 §§1, 5.1, 8.2, 9.3 e 9.5 | Storia, selezione, novità e origini event-based; dichiarazioni ADT invariate |
| 07 §§10.1, 12.4 e 13.4 | Contratti delle fonti, adiacenza e ritorni; distinzione raw/autorevole/certificato |
| 07 §§14.4–14.6 e 15 | Requisiti d’evidenza e matrice di validazione; nuovo §14.6 sulle assunzioni e rinvii |
| 07 §21, esempi 21.4, 21.17, 21.20; §§23.2–23.4 | Binding H e prerequisiti illustrativi allineati; nessuna esecuzione attestata |
| 07 modello finale F2, F3 E/K/T, F4 e F6 | Riepilogo coerente con il corpo; F5 mantiene le garanzie contrattuali, F7 le non-riscritture |
| 06 §§10.4, 12.3–12.6, 14.4, 19.1, 19.4 e nuovi §§28.23–28.25; 07 §§5.1, 7.4, 9.1, 13.3, 15 e modello finale F3 | Correzioni N1/N2/N3 successive alla revisione indipendente di Pro; vincolo sui bordi del gap, quantificazione composizionale e verità del test di prevalenza; dettaglio in §M |
| 08 intero documento | Rigenerato dalla sezione finale del nuovo 07, senza header o testo aggiuntivo |

Parti sostanzialmente preservate: `core.Filter` e contesti autorizzati; catalogo dei 26 operatori e payload; 4 livelli AND/OR e 100 discendenti; budget dei body; algebra V/E/K/P/T; aritmetica terminale; unità, identity e same-occurrence; selezione prima di verify; non-vuoto di All/share; correlazione P; catalogo T; contratti autorevoli e CDE; calendario/ancore/bordi I096; ternarietà; ricette e distinzione analytics/automation; esclusioni/deferred. La fixture 06 §28.20 JSON raw/NativeCurrency e la tabella RF §28.19 sono preservate letteralmente.

## D. Requisiti superflui e requisiti che rimangono

| Requisito 1.1 | Trattamento nella versione 1.2 |
|---|---|
| Certificare che nessun evento reale sia mai mancato dall’acquisizione | Eliminato per il normale dominio event-based |
| Certificare l’inizio della relazione reale per “mai”, primo e novità | Eliminato; tutta la storia registrata interrogabile è sufficiente come dominio |
| Aggiungere occurrence possibili non osservate ai count per storia esterna incompleta | Eliminato; i possibili event-based derivano dai record esistenti e dalle loro incertezze |
| Certificare prefissi/code/periodi esterni per gap, pause, assenze e adiacenza raw | Eliminato; si considerano registrazioni disponibili, senza inventare attività |
| Popolazione e qualificazione delle occurrence esistenti | Preservate: valuta, timestamp, key, dedup e predicati ignoti possono cambiare membership |
| Predecessore del ritorno, ordine e durate | Preservati: entrambe le occurrence devono esistere nel dominio e la pausa deve essere determinabile; niente predecessore al bordo della retention |
| Maturazione e limite incluso della deadline | Preservati: il futuro non è vuoto storico; T=d non matura il negativo |
| Completezza di inventario, stato corrente e lifecycle autorevoli | Preservata: zero eventi non prova zero ticket, cart incompleto o subscription inattiva |
| Garanzie specifiche di named scopes/CDE, first-attainment e fatti storici | Preservate; le primitive non producono né completano automaticamente questi dati |
| Fonte/schema/predicato con significato definito indipendentemente dall’executor | Preservato; nessun nuovo registry per i normali filtri Record |
| Coerenza della vista e riuscita tecnica della lettura | Preservate come precondizioni; implementazione rinviata, errori/parzialità non nascosti come esiti semantici |

## E. Matrice dei cambiamenti di comportamento

Per le righe con un cambiamento, il dato visibile è lo stesso e nella 1.1 manca l’attestazione esterna pertinente. La 1.1 può avere UNKNOWN sul profilo se la capability esiste ma la copertura è insufficiente; per “mai”/primo può essere non attivabile se manca la capacità strutturale di documentare l’inizio. Non si afferma che ogni assenza della 1.1 fosse UNKNOWN: con i suoi prerequisiti completi molte fixture avevano già lo stesso esito.

| Caso concreto | 1.1 | Versione 1.2 | Motivo / fixture |
|---|---|---|---|
| Nessun Purchase visibile in W, nessuna attestazione esterna | Count non necessariamente esatto; assenza può essere UNKNOWN | Count=0, assenza MATCH | Dominio chiuso, CW01 |
| Nessun Purchase nella storia interrogabile, inizio reale non attestato | “Mai” UNKNOWN o non attivabile secondo il prerequisito mancante | “Mai registrato” MATCH | Nuovo significato esplicito, CW02 |
| Nessun utente della sorgente per il profilo, repertorio valido e nessun evento | Non era esplicitato come caso vuoto indipendente dalla copertura | Insieme vuoto, non indisponibilità o UNKNOWN automatici | CW03 |
| Unico Purchase recente EUR e ordinabile, nessun prefisso esterno certificato | First lifetime non necessariamente determinabile | Primo registrato pertinente selezionato, verifica recente/EUR MATCH | CW05 |
| Due soli Export determinati, nessuna attestazione completa esterna | Il terzo può restare UNKNOWN | Terzo e latest 3 NO MATCH | N richiesto invariato, CW05 |
| Ultimo Purchase registrato noto, coda esterna non attestata | Può essere UNKNOWN | Last registrato determinabile | Dati esistenti insufficienti restano U |
| Purchase esistente con NativeCurrency ignota | Nessun EUR UNKNOWN | Nessun EUR UNKNOWN | Il record esiste, CW04 |
| Purchase con JSON null raw confrontato con String EUR | FALSE; assenza raw MATCH nella fixture §28.20 | Invariato | Fixture protetta; governed resta U |
| Refund esistente con order ID ignoto | Assenza del rimborso per O può essere UNKNOWN | Invariato | Non si scarta il record, CW06 |
| Categoria recente, nessuna categoria registrata prima di W | Novità può essere UNKNOWN senza copertura precedente | Nuova rispetto al registrato MATCH; “già vista” NO MATCH | CW13 |
| Prima consegna conservata recente, possibile consegna esterna più antica | Inizio storico non necessariamente determinabile | Primo inizio registrato; niente consegna inventata | CW14; lifecycle certificati distinti |
| Due passi validi e nessun errore/azione intermedia registrati | Assenza/adiacenza può richiedere attestazione della finestra | Assenza/adiacenza sul registrato determinabile | CW16; record intermedio incerto resta U |
| Settimana senza eventi perché non ci sono registrazioni conservate | Poteva essere non osservabile e U | Bucket vuoto, non eliminato | CW15; recognizer applicato normalmente |
| Uso 40 giorni fa, ritorno 5 giorni fa, nessun uso intermedio registrato | Pausa poteva richiedere copertura attestata | Ritorno dopo 35 giorni MATCH | CW08 |
| Solo primo uso registrato recente, predecessore non conservato | Esito dipendeva anche dalla copertura precedente | Nessun ritorno dimostrabile in FromActivity: NO MATCH | Non si inventa u, CW08 |
| Finestra di 60 giorni vuota, max gap≤7 giorni | Può essere U senza copertura | NO MATCH, gap=60 giorni | Regola del vuoto già definita; nessun evento ipotetico |
| Finestra di 5 giorni vuota, max gap≤7 giorni | MATCH: il gap massimo non può comunque superare i 5 giorni della finestra | MATCH, gap=5 giorni sul vuoto registrato | Invariato; prova unilaterale sufficiente e nessuna guardia di non-vuoto |
| Deadline a 7 giorni, consegna 2 giorni fa, zero click | UNKNOWN | UNKNOWN | Futuro non maturato, CW07 |
| Deadline trascorsa, nessun click registrato e nessun record incerto pertinente | Assenza poteva richiedere copertura esterna | Assenza MATCH | CW07; arrivo tardivo può rettificare dopo |
| Retention elimina l’unico Purchase antico | Non si poteva provare “mai nella relazione reale” | Count=0 e mai nella storia interrogabile MATCH | Limite dichiarato, CW09 |
| Inventario Ticket dichiarato incompleto, zero oggetti restituiti | Non prova zero ticket aperti | Invariato | CW17; non è un normale scope raw |
| Producer video assente o fatto storico decisivo ignoto | Capability non attivabile oppure UNKNOWN | Invariato | CW18; nessun falso PASS I051/S012 |
| Evento non ancora attribuito, assente dalla vista coerente | Nessun contratto esecutivo già attestato | Zero nella vista; nessun UNKNOWN automatico per latenza | CW19; garanzia operativa rinviata |
| Lettura realmente fallita o interrotta | Execution failure | Execution failure | CW10, nessun risultato semantico inventato |

La membership può quindi passare da UNKNOWN a MATCH **o a NO MATCH**, secondo il test. Questo riguarda soltanto l’adozione normativa del dominio event-based: non sono cambiamenti implementati nel record filtering. Una cancellazione può anche cambiare una posizione o creare un gap più lungo fra gli eventi rimasti; il risultato non è una ricostruzione del comportamento reale cancellato.

## F. Fixture aggiornate e aggiunte

- **Nuove CW01–CW21:** vuoto/count/mai; sorgente senza utente e repertorio non configurato; valuta ignota; primo/terzo/latest N; identity e timestamp ignoti; deadline; ritorno con predecessore; retention; failure/parzialità; novità; prime consegne; bucket; adiacenza; inventario e producer; latenza; All/None/share e soglie decisive.
- **06 §28.3:** rimosso il vincolo di prefisso reale attestato; conservati tutti i contrasti first/qualifying/in-period.
- **06 §28.9:** pausa riferita a utilizzi registrati, non a certificazione esterna; il predecessore resta obbligatorio.
- **06 §28.12:** l’esempio “tre certi più possibili” ora specifica tre Purchase EUR e un quarto Purchase registrato di valuta ignota; niente occurrence esterne inventate.
- **06 §28.14/O17:** la risposta esistente di classificazione umana/automatica ignota rende esplicita la causa di U, mantenendo inventory/stato/lifecycle e garanzie del contratto.
- **06 §28.15/O15:** settimana incerta con un giorno qualificante certo e un secondo giorno registrato di predicato ignoto; le settimane soltanto vuote non sono U. Gli esiti S..S+U e il denominatore M restano invariati.
- **06 §28.20:** preservata byte per byte, senza cambiare premessa, JSON, tabella, risultati o motivazione. Record raw JSON Null diverso da String EUR resta FALSE; NativeCurrency ignota resta UNKNOWN.
- **07 esempi:** nessun nuovo costruttore; binding storico illustrativo distinto e requisiti di coverage esterna rimossi dagli esempi pertinenti. Gli esiti non interessati sono preservati.

## G. Benchmark: impatto senza nuovi PASS

Il benchmark 03 richiede esplicitamente, nelle convenzioni, la storia completa della relazione reale per “mai” e “primo di sempre”. La versione 1.2 cambia quella premessa di prodotto: **non può essere contabilizzata come un PASS alla premessa originale**. Le tabelle seguenti sono un inventario d’impatto, non una matrice di esecuzione o un nuovo score. Classificazioni 23 CORE, 49 COMMON, 24 ADVANCED e 4 RARE preservate.

### G.1 I001–I100: inventario completo

| ID | Relevance storica | Intento originale | Impatto 1.2 e vincoli preservati |
|---|---|---|---|
| I001 | CORE | Il paese di fatturazione corrente è Italia. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I002 | CORE | Il paese di fatturazione è noto e diverso da Italia. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I003 | CORE | Lavora in un’azienda con 50–249 dipendenti. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I004 | CORE | Si è iscritto negli ultimi 14 giorni. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I005 | CORE | Il dato sulla lingua preferita manca dal profilo. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I006 | COMMON | La lingua preferita è registrata come non nota, con valore null. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I007 | CORE | È attualmente contattabile via email marketing. | **Invariato / policy corrente.** Contattabilità mantiene la policy dichiarata: il mondo chiuso non sostituisce consenso, indirizzo o esclusioni. |
| I008 | COMMON | Fra gli interessi dichiarati compare Sostenibilità. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I009 | CORE | Ha un abbonamento a pagamento attivo oggi. | **Contratto preservato.** Subscription corrente richiede stato pagante e validità autorevoli; zero eventi di avvio non certifica lo stato. |
| I010 | COMMON | La lingua impostata per le comunicazioni differisce da quella preferita dichiarata. | **Invariato.** Proprietà correnti, presenza/null, confrontabilità e requisiti espliciti di valore noto; nessun effetto del mondo chiuso delle occurrence. |
| I011 | CORE | Ha effettuato almeno un Login negli ultimi 7 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I012 | CORE | Non ha acquistato negli ultimi 30 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I013 | CORE | Non ha mai acquistato. | **Storia ridefinita.** “Mai” significa zero Purchase nell’intera storia interrogabile; rimossa la completezza della relazione reale, con limite di retention visibile. |
| I014 | CORE | Ha acquistato almeno 3 volte negli ultimi 90 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I015 | COMMON | Ha acquistato esattamente 3 volte negli ultimi 90 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I016 | COMMON | Ha avviato almeno una volta un abbonamento a pagamento. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I017 | CORE | Negli ultimi 14 giorni ha cliccato la campagna A oppure inviato il modulo Demo. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I018 | CORE | Ha inviato almeno un NPS con voto non superiore a 6 negli ultimi 90 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I019 | CORE | Non ha effettuato alcun acquisto in EUR negli ultimi 90 giorni. | **Assenza ridefinita, valuta preservata.** Zero Purchase registrati prova assenza; un Purchase esistente con NativeCurrency ignota può essere EUR e conserva UNKNOWN. JSON raw resta distinto. |
| I020 | CORE | Ha effettuato almeno un acquisto in una valuta nota diversa da EUR negli ultimi 90 giorni. | **Dominio eventi, noto preservato.** Richiede almeno un acquisto registrato con valuta esplicitamente nota non-EUR. Vuoto e sola valuta ignota non danno un testimone positivo. |
| I021 | COMMON | Negli ultimi 90 giorni ha acquistato almeno una volta e tutti gli acquisti erano in EUR. | **Dominio eventi, All preservato.** Base registrata e non vuota; nessun evento è NO MATCH. Una valuta ignota rimane nel denominatore e può rendere il requisito UNKNOWN. |
| I022 | CORE | Ha fruito di un contenuto Scienza da mobile negli ultimi 30 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I023 | COMMON | Ha fruito di Scienza e ha fruito di contenuti da mobile negli ultimi 30 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I024 | CORE | Ha visitato la landing del lancio fra il 1° e il 31 luglio 2026, con traffico non di test. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I025 | COMMON | Il primo acquisto di sempre è avvenuto negli ultimi 30 giorni ed era in EUR. | **Primo/n-esimo storico ridefinito.** La posizione è nella storia registrata interrogabile; niente certificato d’inizio reale. Base prima di verify, qualificazione, timestamp e ordine restano obbligatori. |
| I026 | COMMON | Il primo acquisto in EUR di sempre è avvenuto negli ultimi 30 giorni. | **Primo/n-esimo storico ridefinito.** La posizione è nella storia registrata interrogabile; niente certificato d’inizio reale. Base prima di verify, qualificazione, timestamp e ordine restano obbligatori. |
| I027 | COMMON | Il primo acquisto degli ultimi 30 giorni era in EUR. | **Posizione nella finestra.** Primo/terzo fra i record della base delimitata; se mancano occurrence sufficienti è NO MATCH, non una posizione completata con eventi ipotetici. |
| I028 | COMMON | Il suo ultimo acquisto è avvenuto negli ultimi 90 giorni, è in EUR e supera €100. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I029 | COMMON | Il suo ultimo acquisto in EUR è avvenuto negli ultimi 90 giorni e supera €100. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I030 | CORE | Ha effettuato almeno un acquisto in EUR superiore a €100 negli ultimi 90 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I031 | COMMON | Il suo ultimo voto NPS è inferiore di almeno 3 punti al precedente. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I032 | ADVANCED | Gli ultimi 3 NPS inviati hanno tutti voto non superiore a 6. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I033 | COMMON | Il terzo utilizzo di Export di sempre è avvenuto negli ultimi 7 giorni. | **Primo/n-esimo storico ridefinito.** La posizione è nella storia registrata interrogabile; niente certificato d’inizio reale. Base prima di verify, qualificazione, timestamp e ordine restano obbligatori. |
| I034 | ADVANCED | Il terzo utilizzo di Export, contando soltanto gli ultimi 30 giorni, è avvenuto negli ultimi 7. | **Posizione nella finestra.** Primo/terzo fra i record della base delimitata; se mancano occurrence sufficienti è NO MATCH, non una posizione completata con eventi ipotetici. |
| I035 | CORE | Ha usato Export per la prima volta entro 7 giorni dall’iscrizione. | **Primo/n-esimo storico ridefinito.** La posizione è nella storia registrata interrogabile; niente certificato d’inizio reale. Base prima di verify, qualificazione, timestamp e ordine restano obbligatori. |
| I036 | COMMON | Ha usato Export nei 7 giorni precedenti il suo ultimo upgrade. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I037 | COMMON | Il primo contatto commerciale dopo la creazione iniziale del lead è arrivato oltre 48 ore dopo. | **Primo seguito registrato.** Primo contatto pertinente registrato dopo l’ancora; zero contatti non significa ritardo infinito. Apertura/creazione iniziale deve mantenere il significato dichiarato. |
| I038 | CORE | Ha speso complessivamente più di €1.000 negli ultimi 90 giorni, prima dei rimborsi. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I039 | COMMON | La spesa netta negli ultimi 90 giorni supera €1.000. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I040 | COMMON | Ha usato almeno 5 funzionalità diverse negli ultimi 30 giorni. | **Identity/grana preservate.** Distinct e ripetizione sui record conservati; zero non richiede storia esterna. Identity ignote o dedup insufficiente restano decisive; unità/righe non diventano ordini. |
| I041 | COMMON | Ha acquistato lo stesso prodotto in almeno 3 ordini distinti negli ultimi 90 giorni. | **Identity/grana preservate.** Distinct e ripetizione sui record conservati; zero non richiede storia esterna. Identity ignote o dedup insufficiente restano decisive; unità/righe non diventano ordini. |
| I042 | COMMON | Ha acquistato almeno 10 unità di articoli negli ultimi 90 giorni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I043 | ADVANCED | Ha rivisto almeno 3 contenuti diversi, ciascuno almeno 2 volte negli ultimi 30 giorni. | **Identity/grana preservate.** Distinct e ripetizione sui record conservati; zero non richiede storia esterna. Identity ignote o dedup insufficiente restano decisive; unità/righe non diventano ordini. |
| I044 | COMMON | Ha guardato video per almeno 120 minuti complessivi negli ultimi 7 giorni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I045 | COMMON | Il valore medio dei suoi acquisti negli ultimi 90 giorni supera €100, con almeno 2 acquisti. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I046 | RARE | È fra i 100 abbonati con più minuti di visione negli ultimi 30 giorni all’interno del proprio piano attuale. | **Fuori dal linguaggio.** Ranking dinamico di popolazione non diventa nativo. La lettura di un ranking pubblicato non implementa l’intento analitico. |
| I047 | COMMON | Ha più acquisti che operazioni di rimborso negli ultimi 90 giorni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I048 | ADVANCED | Ha cliccato entro 7 giorni almeno il 30% delle email consegnate fra 30 e 7 giorni fa, con almeno 10 consegne. | **Origini event-based ridefinite.** Prima consegna registrata per chiave prima di periodo/latest N. Retention può cambiarne l’inizio; restano identity, minimi, attribuzione e scadenza dei singoli esiti. |
| I049 | COMMON | Ha aperto più di un ticket ogni 10 utilizzi del prodotto negli ultimi 30 giorni, con almeno 10 utilizzi. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I050 | COMMON | Ha almeno un ordine degli ultimi 90 giorni interamente rimborsato, anche attraverso più rimborsi parziali. | **Correlazione registrata.** Somma dei rimborsi registrati dello stesso ordine; vuoto vale zero. Pagato, currency/unità e order identity devono restare determinati; nessuna guardia monetaria nuova. |
| I051 | COMMON | Ha visto almeno il 90% di un video negli ultimi 30 giorni, anche in più riprese, senza ricontare parti riviste. | **Conditional invariato.** Producer conforme di copertura unica resta necessario; assenza di eventi raw non certifica soglie, durata o first-attainment. |
| I052 | COMMON | Ha usato Import, Dashboard e Share entro 7 giorni dall’iscrizione. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I053 | ADVANCED | Negli ultimi 30 giorni ha fruito soltanto di contenuti già visti prima del periodo, con almeno una fruizione. | **Presenza nel prefisso registrato.** Già visto prima di W cerca tutte le occurrence precedenti interrogabili. Prefisso vuoto confuta il body; niente completezza reale, niente eliminazione di chiavi ignote. |
| I054 | COMMON | Oltre il 60% delle fruizioni degli ultimi 30 giorni riguarda Scienza, con almeno 5 fruizioni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I055 | ADVANCED | Oltre il 60% del tempo fruito negli ultimi 30 giorni riguarda Scienza, con almeno 60 minuti totali. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I056 | COMMON | Scienza è l’unica categoria più fruita negli ultimi 30 giorni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I057 | ADVANCED | La categoria più fruita è passata da Scienza nei 30 giorni precedenti a Tecnologia nei 30 più recenti. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I058 | ADVANCED | Almeno l’80% delle interazioni marketing degli ultimi 30 giorni avviene fra le 18 e le 22, con almeno 5 interazioni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I059 | RARE | Ha interagito su almeno 3 canali, ciascuno in almeno 2 giorni distinti, negli ultimi 30 giorni. | **Identity/grana preservate.** Distinct e ripetizione sui record conservati; zero non richiede storia esterna. Identity ignote o dedup insufficiente restano decisive; unità/righe non diventano ordini. |
| I060 | ADVANCED | Nessuna categoria supera il 40% delle sue fruizioni negli ultimi 30 giorni, con almeno 10 fruizioni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I061 | COMMON | Negli ultimi 7 giorni ha fruito di una categoria che non aveva mai fruito prima. | **Novità ridefinita.** Nuovo rispetto al prefisso registrato precedente W, anche se vuoto per retention. Non si dichiara nuovo nella vita reale; categoria/tempo ignoti restano pertinenti. |
| I062 | COMMON | Dichiara di preferire email, ma negli ultimi 30 giorni SMS è il canale con più risposte. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I063 | COMMON | Ha usato il prodotto da un paese diverso da quello corrente del profilo negli ultimi 30 giorni. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I064 | CORE | Ha visualizzato un prodotto e ha acquistato negli ultimi 30 giorni. | **Dominio eventi.** Esistenza/count/assenza sulle occurrence registrate nel periodo; zero è determinabile. Restano predicati, same-occurrence, grana e timestamp; dati esistenti decisivi ignoti restano UNKNOWN. |
| I065 | COMMON | Ha visualizzato un prodotto e successivamente ha acquistato, entrambi negli ultimi 30 giorni. | **Percorsi registrati.** Tuple finite fra occurrence disponibili; niente eventi esterni ipotetici. Restano ordine, durata, stessa identity e namespace; non si inferisce stato corrente autorevole. |
| I066 | COMMON | Ha visualizzato un prodotto e poi acquistato quello stesso prodotto, entrambi negli ultimi 30 giorni. | **Percorsi registrati.** Tuple finite fra occurrence disponibili; niente eventi esterni ipotetici. Restano ordine, durata, stessa identity e namespace; non si inferisce stato corrente autorevole. |
| I067 | COMMON | Ha cliccato la campagna A e poi inviato il modulo Demo entro 48 ore, entrambi negli ultimi 14 giorni. | **Percorsi registrati.** Tuple finite fra occurrence disponibili; niente eventi esterni ipotetici. Restano ordine, durata, stessa identity e namespace; non si inferisce stato corrente autorevole. |
| I068 | ADVANCED | Ha inviato il modulo Demo entro 48 ore dall’ultimo click sulla campagna A, entrambi negli ultimi 14 giorni. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I069 | COMMON | Ha creato un progetto, importato dati e condiviso il progetto, in quest’ordine, entro 7 giorni e negli ultimi 30. | **Percorsi registrati.** Tuple finite fra occurrence disponibili; niente eventi esterni ipotetici. Restano ordine, durata, stessa identity e namespace; non si inferisce stato corrente autorevole. |
| I070 | ADVANCED | Ha creato e poi condiviso un progetto negli ultimi 30 giorni senza errori su quel progetto fra le due azioni. | **Assenza/adiacenza registrata.** Nessun errore/azione intermedia registrati negli slot autorizzati. Nessuna completezza esterna; eventi intermedi esistenti di qualificazione/tempo incerti restano possibili. |
| I071 | ADVANCED | Negli ultimi 30 giorni ha contattato l’assistenza come prima azione dopo un errore, entro 10 minuti. | **Assenza/adiacenza registrata.** Nessun errore/azione intermedia registrati negli slot autorizzati. Nessuna completezza esterna; eventi intermedi esistenti di qualificazione/tempo incerti restano possibili. |
| I072 | COMMON | Ha ricevuto un’email Demo fra 30 e 7 giorni fa senza cliccarla entro i 7 giorni successivi alla consegna. | **Assenza registrata / maturazione.** Nessun click registrato pertinente; entro sette giorni resta distinto da UntilT. Deadline futura o al bordo T=d resta UNKNOWN; click tardivo può rettificare la vista successiva. |
| I073 | COMMON | Ha ricevuto un’email Demo fra 30 e 7 giorni fa e non l’ha ancora cliccata. | **Assenza registrata / maturazione.** Nessun click registrato pertinente; entro sette giorni resta distinto da UntilT. Deadline futura o al bordo T=d resta UNKNOWN; click tardivo può rettificare la vista successiva. |
| I074 | CORE | Ha un carrello aggiornato fra 30 giorni e 24 ore fa che non è stato completato. | **Contratto preservato.** Cart/Ticket richiedono stato e lifecycle autorevoli. Nessun evento raw non prova non-completamento o assenza di ticket aperti. |
| I075 | CORE | Ha almeno un ticket tuttora aperto, anche se altri sono stati chiusi. | **Contratto preservato.** Cart/Ticket richiedono stato e lifecycle autorevoli. Nessun evento raw non prova non-completamento o assenza di ticket aperti. |
| I076 | COMMON | Ha almeno un acquisto degli ultimi 90 giorni senza alcun rimborso per quell’ordine fino a oggi. | **Assenza correlata registrata.** Rimborsi del medesimo ordine nel patrimonio disponibile. Alcuni vs tutti non vacuo invariato; Refund esistente con key ignota non si scarta. |
| I077 | COMMON | Ha acquistato negli ultimi 90 giorni e nessuno degli ordini acquistati nel periodo è stato rimborsato fino a oggi. | **Assenza correlata registrata.** Rimborsi del medesimo ordine nel patrimonio disponibile. Alcuni vs tutti non vacuo invariato; Refund esistente con key ignota non si scarta. |
| I078 | ADVANCED | Tutti i ticket aperti fra 90 giorni e 48 ore fa hanno ricevuto una prima risposta entro 48 ore, con almeno 3 ticket. | **Opportunità e risposta.** Esiti sulle risposte registrate con deadline e minimo tre; restano ruolo umano/automatico, identity e apertura iniziale/lifecycle dichiarati. |
| I079 | ADVANCED | Lo stesso ticket è stato riaperto almeno 2 volte dopo altrettante chiusure negli ultimi 90 giorni. | **Percorsi registrati.** Tuple finite fra occurrence disponibili; niente eventi esterni ipotetici. Restano ordine, durata, stessa identity e namespace; non si inferisce stato corrente autorevole. |
| I080 | COMMON | Ha usato il prodotto in almeno 4 giorni distinti nelle ultime 4 settimane complete. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I081 | COMMON | Ha usato il prodotto in ognuna delle ultime 4 settimane complete. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I082 | ADVANCED | Ha usato il prodotto in almeno 3 giorni di ognuna delle ultime 4 settimane complete. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I083 | ADVANCED | Ha avviato almeno 5 video in un qualsiasi intervallo continuo di 24 ore contenuto negli ultimi 7 giorni completi. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I084 | COMMON | Ha avviato almeno 5 video nello stesso giorno di calendario negli ultimi 7 giorni completi. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I085 | ADVANCED | Ha usato il prodotto ogni giorno per almeno 7 giorni consecutivi, in qualche punto degli ultimi 30 giorni completi. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I086 | ADVANCED | Negli ultimi 60 giorni non ha mai lasciato passare più di 7 giorni senza usare il prodotto. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I087 | COMMON | Gli utilizzi negli ultimi 14 giorni sono diminuiti di almeno il 50% rispetto ai 14 precedenti, nei quali erano almeno 10. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I088 | ADVANCED | Il numero di utilizzi è diminuito da una settimana alla successiva nelle ultime 3 settimane complete, con almeno 10 nella prima. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I089 | COMMON | Usava Export nei 14 giorni precedenti, non lo usa nei 14 più recenti, ma usa altre funzionalità almeno 3 volte nello stesso periodo recente. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I090 | COMMON | È tornato a usare il prodotto negli ultimi 7 giorni dopo almeno 30 giorni senza utilizzi, e lo aveva già usato prima della pausa. | **Ritorni registrati.** Predecessore e ritorno devono essere registrati e consecutivi. Nessuna attestazione della pausa reale; primo registrato non è un ritorno e retention non inventa il predecessore. |
| I091 | COMMON | Non ha usato il prodotto nell’ultima settimana completa, dopo averlo usato in ciascuna delle 4 settimane precedenti. | **Serie registrata.** Bucket senza eventi vuoti, non UNKNOWN da mancata storia esterna. Serie/calendario, distinti, burst, streak e gap ai bordi invariati; dati esistenti incerti restano pertinenti. |
| I092 | RARE | Negli ultimi 180 giorni ha alternato almeno 3 volte uso, una pausa di almeno 14 giorni e ritorno. | **Ritorni registrati.** Predecessore e ritorno devono essere registrati e consecutivi. Nessuna attestazione della pausa reale; primo registrato non è un ritorno e retention non inventa il predecessore. |
| I093 | ADVANCED | Fra gli ultimi 3 acquisti, l’intervallo più recente è almeno il doppio di quello precedente. | **Selezione registrata.** Ultimo/blocco/preset sui dati registrati identificabili; nessuna attestazione della coda esterna. Restano N richiesto, confronto autorizzato, ordine e divieto di fallback. |
| I094 | RARE | Non acquista da più del doppio del suo intervallo mediano fra acquisti, misurato sugli ultimi 5 acquisti. | **Deferred preservato.** Mediana/cadenza personale non inclusa; il mondo chiuso non aggiunge statistiche o un producer. |
| I095 | COMMON | Ha consumato oltre l’80% della quota del piano attuale nel proprio ciclo di fatturazione corrente. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I096 | ADVANCED | Ha fatto l’ultimo upgrade fra 90 e 14 giorni fa e ha più che raddoppiato gli utilizzi nei 14 giorni successivi rispetto ai 14 precedenti, con almeno 10 prima. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |
| I097 | ADVANCED | Ha completato tutti i materiali formativi assegnati nel proprio percorso corrente. | **Deferred preservato.** Completamento dell’insieme assegnato non incluso; inventario, non-vuoto e identity non sono sostituiti da count di eventi. |
| I098 | ADVANCED | Negli ultimi 30 giorni gli è stato inviato marketing su un canale non autorizzato al momento dell’invio. | **Deferred/conditional preservati.** Ricostruzione da cronologia resta deferred; snapshot storico già certificato consumabile. Current consent non è autorizzazione all’invio passato. |
| I099 | ADVANCED | Ha cliccato almeno 2 delle ultime 3 campagne distinte ricevute. | **Origini event-based ridefinite.** Prima consegna registrata per chiave prima di periodo/latest N. Retention può cambiarne l’inizio; restano identity, minimi, attribuzione e scadenza dei singoli esiti. |
| I100 | COMMON | Negli ultimi 30 giorni gli è stato inviato marketing su un canale che oggi non è autorizzato. | **Misure/scope registrati.** Popolazione di eventi registrati; nessun evento esterno ipotetico. Restano unità, campioni/denominatori, valori ignoti, proprietà correnti, calendari e stadi di selezione richiesti. |

### G.2 S001–S015: inventario completo

Ogni segmento mantiene tutte le proprie congiunzioni e dipendenze. Determinare una sola assenza a mondo chiuso non basta a dichiarare supportato il segmento completo.

| ID | Segmento | Impatto e dipendenze |
|---|---|---|
| S001 | Clienti abituali da recuperare | **Dominio eventi.** Count/spesa e assenze sul registrato; contattabilità, unità e minimi invariati. |
| S002 | Carrelli con interesse ripetuto | **Contratto Cart preservato.** View registrate del prodotto membro; inventario/contenuto corrente, vero ultimo aggiornamento e lifecycle non sono dedotti dal vuoto raw. |
| S003 | Acquirenti pronti al cross-selling | **“Mai” ridefinito.** Assenza di Accessori nell’intera storia registrata; ordine senza rimborsi sul registrato con key sufficiente. Non attesta assenza di acquisti esterni. |
| S004 | Trial iniziati ma non decollati | **Contratto Trial preservato.** Import/Dashboard/Share e Upgrade sul registrato negli slot distinti. Trial corrente e associazione alle attività restano certificati; minimi e quattro scope invariati. |
| S005 | Candidati a un piano superiore | **Dominio eventi e fatti personali.** Bucket di utilizzo e assenza Upgrade sul registrato; quota/ciclo attuali e unità restano necessari, senza inventare settimane attive. |
| S006 | Funzionalità persa, relazione ancora attiva | **Dominio eventi + Ticket.** Export/altro uso sul registrato; zero ticket aperti richiede inventario/stato autorevoli. |
| S007 | Lead caldi senza presa in carico | **Assenza registrata.** Zero contatti registrati dalla creazione; creazione iniziale e proprietà correnti devono avere il significato dichiarato. |
| S008 | Ex clienti inattivi ma ancora ricettivi | **Opportunità e stato.** Uso e assenza sul registrato; ultime tre campagne per prima consegna registrata. Stato corrente/ex-pagante e contattabilità restano distinti. |
| S009 | Destinatari da alleggerire | **Assenze correlate registrate.** Almeno cinque messaggi consegnati distinti; nessun esito registrato pertinente fino a T. Un tentativo di invio non diventa consegna. |
| S010 | Lettori tematici da convertire | **“Mai avviato” ridefinito.** Nessun avvio pagante registrato nella storia interrogabile; stato gratuito corrente, share e contattabilità non sono inferiti da quella sola assenza. |
| S011 | Spettatori appena rientrati | **Ritorno registrato + Subscription.** Pausa e campagne sul registrato con predecessore; minuti recenti indipendenti. Subscription attiva/contattabilità restano richieste. |
| S012 | Serie lasciata incompiuta | **Conditional preservato.** Assenza PlaybackStarted dopo milestone sul registrato; producer 90%, first-attainment, disponibilità storica e Subscription restano requisiti separati. |
| S013 | Rinnovi da seguire con priorità | **Dominio eventi + Ticket.** Ultimo NPS e calo fra periodi sul registrato; campione minimo, rinnovo/piano attuali e ticket aperto autorevole invariati. |
| S014 | Ticket chiusi, esperienza non ripresa | **Dominio eventi + stato.** Ultima chiusura e utilizzi prima/dopo sul registrato; zero ticket aperti e stato pagante richiedono autorità corrente, non saldi di eventi. |
| S015 | Clienti pronti per un programma di testimonianze | **Deferred preservato.** I097 mantiene S015 fuori dalla copertura completa. Il mondo chiuso su attività/escalation non fornisce inventario assegnati, completamenti o stato autorevole. |

### G.3 Out-of-sample O01–O20: disponibilità e limiti

Il file originario `intent_out_of_sample_fissati(1).md` è richiamato dalle versioni storiche, ma non è presente fra le fonti disponibili nel repository. I 20 small hard cases del benchmark 03 sono ID I già inclusi nei 100: **non sono** il corpus O01–O20. Si verificano soltanto i richiami effettivamente contenuti in 06/07, senza ricostruire testi mancanti o dichiarare PASS integrali.

| ID | Evidenza disponibile e conclusione |
|---|---|
| O01 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O02 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O03 | Fixture esplicite 06 §28.13: primo Purchase registrato prima del path, blocco order sui passi 1–2 e terzo esterno. Cambia solo la pretesa di storia esterna completa; ownership/identity restano. |
| O04 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O05 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O06 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O07 | Richiamo parziale in 06 §28.16: confronto medie e requisito distinto sugli errori. Popolazione registrata; nessuna ricostruzione o verifica dell’intero intento mancante. |
| O08 | Esclusione esplicita: attività di altri membri account non esprimibile sui soli eventi individuali/relazioni raw. Resta fuori dal linguaggio. |
| O09 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O10 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O11 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O12 | Richiamo parziale in 06 §26.4: propensity score pubblicato consumabile con contratto; modello predittivo non nativo. Nessun nuovo PASS al calcolo. |
| O13 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O14 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O15 | Fixture esplicite 06 §28.15: QPC con quattro bucket fissi. Vuoto distinto da dati esistenti incerti; nessun auto-nesting o nuova capacità. |
| O16 | Richiami in 06 §§26.4 e 28.18: ranking pubblicato consumabile; ranking dinamico non nativo. Nessun nuovo PASS al calcolo. |
| O17 | Fixture esplicite 06 §28.14: Ticket corrente, apertura iniziale >48h e Human Response same-lifecycle. Contratto e garanzie restano; U esemplificato con risposta esistente non classificabile. |
| O18 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O19 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |
| O20 | Testo dell’intento non disponibile; nessuna conclusione specifica o nuovo PASS. Si applicano soltanto le regole generali e le esclusioni già definite. |

**Controlli contro falsi PASS:** I046 resta fuori dal linguaggio; I094, I097/S015 e ricostruzione I098 restano deferred; I051/S012 mantengono producer/contratti; O08 resta escluso; O12/O16 non ricevono credito di computazione nativa. Nessuna percentuale di copertura ricalcolata e nessun esito runtime dichiarato.

## H. Tre rinvii operativi, non soluzioni implementate

| Lavoro | Assunzione per progettare adesso | Stato e confine |
|---|---|---|
| Identity Resolution / pubblicazione | Vista coerente di profili, associazioni ed evidenze fornita alla query | Associazione aggiornata prima della vista Profiles resta un rischio reale descritto dal rapporto 11; soluzione autonoma rinviata, nessun protocollo obbligatorio deciso |
| Writer asincrono / attribuzione KPID | Il dominio è la vista coerente degli eventi disponibili attribuiti, non la coda degli eventi accettati | Latenza non produce automaticamente U; niente watermark/certificato obbligatorio nel linguaggio. Failure/parzialità resta errore. Implementazione e verifica di acquisizione/coerenza rinviate |
| Destination matching | Il sottosistema export fornisce una vista valida per la selezione operativa | Cancellazione/ricarica per batch resta da correggere separatamente; consensi, matching e modalità non fanno parte della membership e non impongono protocolli al compilatore |

Questi lavori non bloccano il progetto corrente del compilatore e del piano fisico. Possono essere prerequisiti di rilascio delle funzionalità dipendenti; nessuna funzionalità va dichiarata correttamente rilasciabile sulla sola assunzione documentale.

## I. Conseguenze da applicare successivamente alla proposta architetturale 12

Il file 12 resta intatto e non approvato. Con l’approvazione normativa della 1.2, la sua revisione architetturale separata dovrà considerare almeno i seguenti punti:

| Parte del 12 | Conseguenza della versione 1.2 |
|---|---|
| §§3.2 e 4.3, evidenza e pianificazione | Eliminare count/candidati possibili aggiunti per eventi esterni mai registrati. Conservare certezza/possibilità dei record esistenti e garanzie autorevoli/CDE |
| §4.1, campi ReadView | Generazioni, ingestion/attribution cuts, certificates e retainedResultHandle non sono obblighi normativi del filtering. Restano alternative tecniche, non contratti approvati |
| §§5.1–5.4, coerenza e acquisizione | Distinguere l’assunzione di vista coerente dai lavori autonomi rinviati; nessun coordinamento/generazione/watermark imposto al compilatore |
| §§5.5–5.6, snapshot e risultati conservati | Snapshot, Time Travel, materializzazione e durata restano scelte future da confrontare, non prerequisiti semantici introdotti dalla 1.2 |
| §§6.5 e 8.6, export | Assumere matching valido fornito dalla destination, senza protocollo a carico di Unified. Mantenere membership separata dalla selezione operativa |
| §§8.1–8.5, prova di sufficienza | Count/assenza su registrato, primo disponibile pertinente, novità e ritorni senza certificazione esterna. Preservare la prova NativeCurrency e le incertezze di record esistenti |
| §§9–10, dipendenze e decisioni | Spostare i tre lavori operativi su percorsi autonomi e distinguere prerequisiti di rilascio dalle decisioni del compilatore/piano fisico |

Il principio di un significato V/E/K/P/T comune ai due executor rimane coerente con la baseline semantica 1.2, ma nessuna sua architettura concreta viene approvata da questa revisione.

## J. Approvazione registrata e verifiche ancora necessarie

1. **Approvazione della versione 1.2 registrata:** sono approvati il mondo chiuso degli eventi, inclusi gli effetti visibili di retention su primo, novità, prime consegne e ritorni, e le correzioni N1/N2/N3 di §M. N2 è confermata esplicitamente: quantificazione composizionale dei risultati ternari individuali K senza inferenza globale fra candidati distinti, preservando tutte le dipendenze interne obbligatorie. Non sono policy da scegliere per singolo filtro o executor.
2. **Adozione di significati versionati:** preservare identity/revisioni già immutabili e distinguere eventuali documenti/result cache Unified precedenti. Formato/migrazione non sono progettati qui; nessun cambiamento ai filtri Record installati.
3. **Presentazione del limite storico:** definire nella progettazione UI la formulazione di “mai/primo registrato” e dei limiti noti di retention, evitando promesse sulla vita reale. La scelta delle etichette non può cambiare gli esiti normativi.
4. **Architettura Fase 4B:** contratto fisico, esecuzione, gestione dei risultati e durata restano da approvare separatamente; il rapporto 12 non è adottato.
5. **Rilascio:** implementare e verificare separatamente coerenza IR/acquisizione e validità del matching quando richieste dalla funzionalità; restano irrisolte allo stato del codice.
6. **Copertura documentale mancante:** recuperare l’allegato O01–O20 per una review integrale di quegli intenti; non inventare risultati per i testi non disponibili. Il presente rapporto verifica i richiami effettivi e conserva tutte le esclusioni.

## K. Verifiche documentali della prima stesura (storico)

Controlli della prima stesura su copie temporanee e sui quattro output. La successiva revisione ha individuato i residui corretti e verificati in §L; questi controlli iniziali non attestavano l’assenza di ogni residuo:

- 08 coincide byte per byte con la sezione finale di 07 dal titolo `# Canonical AST — Phase 4A Final Model` fino a EOF, senza prefazioni aggiuntive.
- Le dichiarazioni ADT/grammatica del 07 sono invariate. Confrontando tutti i blocchi di codice, l’unica variazione è il binding illustrativo H; nessun nuovo costruttore o payload.
- Fixture 06 §28.20 e raccordo RF §28.19 preservati letteralmente; catalogo §8.1, contratti §15.6–15.7, CDE §16, tempo/I096 §17 e composizione ternaria §20 verificati invariati.
- Tutti i 100 ID I e i 15 ID S sono presenti una sola volta nell’inventario d’impatto; classificazioni storiche preservate. O01–O20 inventariati distinguendo contenuto disponibile e mancante.
- CW01–CW21 sono distinti e includono tutti i casi richiesti; sono fixture normative, non test runtime.
- Verificati titoli/rinvii delle sezioni interessate, link locali, chiusura dei blocchi Markdown e stato di proposta. Nella prima stesura erano rimasti la riga obbligatoria “Tempo e copertura” di 06 §22.1 e alcuni rimandi correnti alla 1.1; sono corretti in §L. Le prescrizioni su lifecycle/contratti/CDE restano.
- Checksum degli input preesistenti confrontati prima/dopo: originali 1.1, benchmark 03, audit, rapporti 11/12 e tutti gli altri file preesistenti in filtering invariati. File tracciati e staging invariati; soltanto i quattro nuovi documenti sono aggiunti come file non tracciati.

Comandi di verifica: script Python temporanei `revise.py`, `report.py` e `verify.py`; `git rev-parse HEAD`, `git status --short`, `git diff --stat`, `git diff --cached --stat`. Non sono stati eseguiti test applicativi o accessi a warehouse. Nessun commit.

### K.1 Impronte degli input principali preservati

| File | SHA-256 |
|---|---|
| 03-benchmark-intenti-segmentazione-v1.0.md | `8291a0938c9f32f0f040ee0d1c836d8b2ae6be8569f68b77f844a75c288ff01c` |
| 06-specifica-semantica-riconciliata-v1.1.md | `0453b94d36925e57a8c89ab1a7dd0a6a400fb22d56678326cf6c02748ba1a817` |
| 07-ast-canonico-riconciliato-v1.1.md | `b50885a13c5c670f2b100dfe7080b2be3ef1f4f77a6362051d82b3283fa2f833` |
| 08-ast-canonico-estratto-v1.1.md | `d96fdf20cbece06d323730b3932d7ae4da8dc70bc036a0d29f8206d4f0202ccb` |
| 11-ricognizione-architetturale-filtering-warehouse-fase-4b.md | `b2961e3590adb3efba12f1aa976b5de05e5580c1c000097ab77890e6199ea11f` |
| 12-proposta-architettura-interrogazioni-unified-fase-4b.md | `97cf5f2d9e2c25f81e57efd138251b49ad534bafebf302821335c3d4907108bf` |

## L. Correzioni documentali mirate prima della revisione indipendente (storico)

**Fase storica, precedente alla revisione indipendente e all’approvazione definitiva.** Le correzioni seguenti precisarono la decisione allora proposta senza cambiare gli esiti concordati, il linguaggio o l’architettura.

| Richiesta | Correzione effettuata |
|---|---|
| 06 §22.1 — Tempo e copertura | Per i normali eventi registrati richiede timestamp, ordine quando necessario, repertorio, grana e limiti noti della storia interrogabile. Le ulteriori garanzie storiche/di completezza restano soltanto quelle previste da contratti autorevoli e CDE; eliminato il requisito generalizzato sull’inizio della relazione reale |
| 06 §28.1 — Premesse degli acceptance | “Periodo completamente osservato”, “osservazione completa” e analoghe premesse event-based si riferiscono al dataset della vista interrogabile e ai dati dichiarati dal test, compresi gli stati ignoti espliciti. Non certificano eventi reali esterni, non impongono un inizio attestato e non rendono nota NativeCurrency |
| 06 §28.20 — Fixture protetta | Testo, JSON, premesse, tabella ed esiti interamente invariati. Il chiarimento è soltanto nelle premesse generali §28.1: raw JSON Null vs String EUR rimane FALSE e assenza raw MATCH; NativeCurrency sconosciuta rimane UNKNOWN |
| 06 §20.1 e 07 §19 — Versione corrente | Riferimenti normativi formulati per la versione 1.2. L’origine dei limiti nella 1.1 rimane un richiamo storico esplicito; nessuna nuova approvazione della 1.2 o variazione dei limiti |
| 07 §§5.3, 7.5 e 19 — Copertura nei riepiloghi | “Copertura” per deadline e normali mapping riferita esplicitamente ai dati registrati pertinenti, mantenendo T>d, estremo incluso e garanzie ulteriori dei contratti. Nessuna attestazione esterna generalizzata |
| FullRelevantHistory — 06 §22.3; 07 §5.1 e F2 | Risoluzione sulle definizioni governate pertinenti della fonte/repertorio. Se cambia il significato della definizione 1.1, occorre una nuova identità o meaningRevision immutabile e adozione esplicita; etichette/alias non reinterpretano la stessa revisione. Nessun costruttore AST o DefinitionKind nuovo, nessun registry aggiuntivo per Record |
| 08 | Rigenerato come estratto esatto dell’intera sezione finale del 07 aggiornato |

**Verifiche della correzione:**

- Rilette le prescrizioni di tempo, osservazione, copertura, storia e revisione nei documenti proposti. Nei normali percorsi event-based non resta obbligatoria un’attestazione dell’inizio reale o della completezza degli eventi esterni. I richiami residui alla 1.1 sono storici intenzionali o riferimenti alla baseline 1.1 allora vigente.
- Confronto byte per byte di 08 con la sezione finale di 07; nessuna aggiunta autonoma all’estratto.
- Tutti i 94 blocchi di codice del 07 identici alla proposta precedente: costruttori, payload e binding illustrativo H invariati in questa correzione.
- Fixture 06 §28.20, raccordo Record §28.19, 21 fixture CW, catalogo dei 26 operatori, deadline/tempo/I096 e garanzie di contratti autorevoli/CDE invariati. Tabella ternaria e regole di UNKNOWN preservate; in §20.1 cambia soltanto il riferimento alla versione corrente.
- Confronto degli hash di tutti i file preesistenti in filtering esclusi i quattro output: documenti 1.1, benchmark 03, rapporti 11/12 e altri documenti intatti. Nessun file aggiunto o rimosso; codice tracciato e staging invariati, HEAD ancora `25f639d84eb35815af5d1062c86545b65b97d720`.
- Controlli eseguiti mediante script Python temporanei `correct.py` e `verify-corrections.py`, lettura dei diff e comandi Git di sola verifica. Nessun test applicativo o accesso a warehouse, nessun commit, nessuna implementazione della Fase 4B.

Rimangono i rinvii operativi, le alternative architetturali non approvate e il limite sui testi out-of-sample mancanti già dichiarati in questo rapporto. Non vengono introdotte nuove decisioni di prodotto o policy di UNKNOWN.

## M. Correzioni normative dopo la revisione indipendente di Pro (storico della revisione)

**Fase storica della revisione, conclusa con l’approvazione definitiva di §N.** La revisione indipendente confermò il mondo chiuso degli eventi e richiese le correzioni N1/N2/N3. Queste furono recepite prima dell’approvazione, senza demandarne la scelta agli executor. Le sezioni K e L descrivono le verifiche delle stesure precedenti; i controlli qui sotto riguardano la correzione successiva a Pro.

### M.1 N1 — MaximumGap e periodo con estremi finiti

MaximumGap misura anche il vuoto iniziale e finale; non è definibile dalla sola sequenza degli eventi. 06 §14.4 e 07 §§5.1, 13.3, 15 e F3/T richiedono un periodo con estremi inferiore e superiore finiti, semanticamente previsti dalla forma. Il catalogo 06 §14.1 e l’indice §30 sono allineati. Non si modifica l’ADT: il requisito è verificato sui parametri e sulle definizioni dei periodi già presenti.

| Forma / situazione | Regola approvata |
|---|---|
| FullRelevantHistory da solo | Invalido per MaximumGap anche se esiste un primo evento registrato. La storia interrogabile non fornisce necessariamente un inizio finito |
| AvailableHistory | Ammesso se la definizione risolta dichiara un intervallo con entrambi gli estremi finiti. Sola policy di retention o prima riga disponibile non li sostituiscono |
| Assoluto, elapsed, calendar o personale delimitati e già autorizzati per lo scope | Ammessi con i propri controlli. Non si aggiungono nuove finestre o ancore al costruttore |
| Estremo strutturalmente mancante | Configurazione invalida: InvalidShape per la forma, InvalidAgainstRegistry per definizione risolta incompatibile. Non UNKNOWN o capability indisponibile |
| Estremo previsto ma valore ignoto sul profilo | Configurazione valida; UNKNOWN quando il valore è decisivo. Non si inventa il bordo |
| Ciclo corrente dinamico con inizio=T | Il solo vuoto non è invalidità: gap=0. Resta start<end per un intervallo assoluto immesso |

**Dipendenze temporali analoghe verificate:** QPC ha già una serie finita di periodi calendar completi e il max gap usa i bordi del bucket. Rolling burst usa finestre finite interamente nel periodo; calendar burst e streak usano giorni finiti e test esistenziali: non necessitano del vuoto iniziale/finale dell’intera storia, quindi nessun nuovo divieto su FullRelevantHistory. I ritorni richiedono u/r registrati, consecutività e pausa determinabile; EntirePauseIn mantiene il proprio BoundedObservationPeriod, senza sostituire il predecessore con il bordo storico. Le assenze P/deadline hanno già i propri estremi e maturazione. La novità K usa il bordo iniziale del periodo recente già richiesto, senza pretendere un inizio finito della storia precedente. Non sono introdotti altri divieti o attestati esterni.

Nuove fixture **N1-01–N1-07** in 06 §28.23: storia intera invalida per il gap, AvailableHistory delimitata, definizione senza bordo, bordo personale ignoto decisivo, assoluto/lookback, ciclo inizialmente vuoto e conservazione dei confini QPC/burst/streak.

### M.2 N2 — K composizionale, senza inferenza globale fra candidati

06 §§12.3 e 12.6 e 07 §9.1/F3-K fissano la quantificazione dei risultati ternari individuali su una base determinata: D=S+F+U, count nell’inviluppo [S,S+U], con le regole di Some, None, Cardinality, All e Share e le rispettive guardie. Il quantificatore non restringe l’inviluppo usando dipendenze fra candidati distinti. L’inviluppo è conservativo e può includere conteggi non globalmente realizzabili; non afferma che tali conteggi siano possibili né che i candidati siano indipendenti nel dominio.

La nuova **fixture N2-01**, 06 §28.24, ha due ordini certi O1/O2 e un solo Refund con orderId ignoto ma univalore. Ogni body “quest’ordine non ha rimborsi” è UNKNOWN; Some è UNKNOWN con S=F=0, U=2. Una prova globale potrebbe dimostrare che almeno un ordine non è rimborsato. La versione 1.2 esclude esplicitamente quell’inferenza nella quantificazione K: non la dichiara logicamente impossibile e non lascia la scelta all’executor. Questa limitazione, inizialmente proposta nella revisione, è una **scelta normativa approvata esplicitamente nella versione 1.2** (§N).

06 §19.1 chiarisce che UNKNOWN è relativo alle regole di evidenza/composizione previste; §19.4 distingue il confronto interno alle costruzioni dalla quantificazione composizionale dei risultati K. Sono allineati la matrice 07 §15, il delta §23.4 e il modello finale. Non si applica indiscriminatamente una regola di indipendenza: restano obbligatorie le selezioni compatibili di E, la chiave comune all’intero blocco P, part/whole e denominatori/pesi condivisi delle share, le guardie e l’ownership. Una base incerta non viene dichiarata determinata per applicare S/F/U. Non cambia il contratto di inventari autorevoli e CDE.

### M.3 N3 — Rilevanza degli ignoti sulla verità del test

06 §§10.4 e 19.4 e 07 §7.4, matrice §15 e F3/V richiedono UNKNOWN quando l’ignoto può cambiare la verità del test completo richiesto, guardie incluse. La sola incertezza sull’identità del vincitore non basta. I test rimangono terminali; non si inventano una categoria “sconosciuto”, un vincitore o un tie-break.

Le nuove fixture **N3-01–N3-04**, 06 §28.25, verificano: Scienza=0/Arte=10/Tecnologia=10 e una categoria ignota → “Scienza unica prevalente” NO MATCH; stessi dati con Different Scienza → UNKNOWN per la possibile unicità; Scienza/Arte a 10 e una categoria ignota → Equal Scienza UNKNOWN; parità massima completamente nota → Different NO MATCH per la guardia falsa. Unicità, massa ignota e minimo campione sono preservati.

### M.4 Matrice delle differenze introdotte da questa correzione

| Caso | Formulazione precedente della versione 1.2 | Versione 1.2 corretta | Motivo |
|---|---|---|---|
| Gap su FullRelevantHistory | Formula con s/e, senza obbligo esplicito dei due bordi; rischio di usare primo evento o retention | Configurazione invalida per MaximumGap da solo | Occorre misurare anche il vuoto iniziale e finale |
| Gap su AvailableHistory delimitata e nota, senza eventi | Ammesso ma non distinto esplicitamente dal caso precedente | Ammesso; gap=durata dichiarata | Entrambi i bordi sono forniti dal periodo, non inferiti dagli eventi |
| Bordo personale dichiarato ma ignoto | Mancava la distinzione specifica per il gap | Possibile UNKNOWN quando decisivo; non invalidità della forma | Dato ignoto distinto da struttura priva di un estremo |
| Due ordini/un solo rimborso a chiave ignota | §12.6 indicava S/U ma §19.4 poteva richiedere una prova globale sui valori compatibili | Body UNKNOWN/UNKNOWN; Some UNKNOWN, senza inferenza globale | Scelta composizionale esplicita e uniforme fra executor |
| Scienza 0, Arte/Tecnologia 10, un ignoto | “Può cambiare il vincitore” poteva imporre UNKNOWN | Equal Scienza NO MATCH; Different Scienza UNKNOWN | Si valuta la verità del test richiesto, non la sola identità del vincitore |

### M.5 Controlli di questa revisione e limiti

- Riletti i diff delle quattro proposte e controllati i rinvii alle sezioni N1/N2/N3, incluse le formulazioni generali, le matrici e il modello finale. I nuovi esempi sono fixture normative di progetto, non risultati runtime.
- 08 rigenerato e confrontato byte per byte con la sezione finale di 07, dal titolo `# Canonical AST — Phase 4A Final Model` fino a EOF.
- Tutti i 94 blocchi di codice del 07 invariati rispetto alla proposta precedente: nessun nuovo costruttore, campo, payload o flag. Controllati limiti AND/OR e budget dei body, catalogo Record e contratto core.Filter.
- Fixture 06 §§28.19–28.20 interamente preservate; mondo chiuso, retention, 21 fixture CW, deadline, contratti autorevoli/CDE e i tre rinvii infrastrutturali preservati. Nessuna modifica ai testi del benchmark e nessun nuovo PASS/deferred promosso.
- Controllati gli esiti delle nuove fixture e le regole S/F/U su base determinata con verifiche documentali e calcoli su copie temporanee. Questa verifica non implementa né esegue i filtri Unified.
- Verificati link locali, esistenza delle sezioni richiamate, chiusura dei blocchi Markdown e stato di proposta. Tutti i documenti preesistenti in filtering esclusi i quattro output sono confrontati tramite SHA-256 e rimangono identici, inclusi 1.1, benchmark 03 e rapporti 11/12.
- Comandi di sola verifica Git e script Python temporanei `correct-pro.py` / `verify-pro.py`. Nessun test Go/PostgreSQL/Snowflake o accesso a warehouse, nessuna modifica applicativa, nessuna implementazione della Fase 4B, nessun commit o staging.

L’approvazione esplicita della 1.2 corretta, in particolare della scelta N2, è registrata in §N. Restano da affrontare le decisioni architetturali future e i lavori infrastrutturali autonomi già rinviati. Il rapporto 12 non è aggiornato né adottato.

## N. Approvazione e consolidamento definitivo della versione 1.2

### N.1 Nuova baseline e decisioni confermate

Il titolare del progetto ha approvato formalmente la versione 1.2, comprese le correzioni N1/N2/N3. I documenti [06](06-specifica-semantica-riconciliata-v1.2.md), [07](07-ast-canonico-riconciliato-v1.2.md) e [08](08-ast-canonico-estratto-v1.2.md) costituiscono la nuova baseline normativa e tecnica. Questo rapporto è definitivo e conserva analisi, matrici, revisioni e verifiche precedenti. Le versioni approvate 1.1 restano intatte come riferimento storico.

- **N1:** MaximumGap richiede estremi inferiore e superiore finiti previsti dalla forma temporale. FullRelevantHistory da solo non è ammissibile; AvailableHistory è ammesso con un intervallo dichiarato completo di estremi. Estremo strutturalmente assente significa configurazione invalida; valore dell’estremo previsto ma ignoto sul profilo può dare UNKNOWN quando decisivo. Nessun nuovo divieto per burst, streak o QPC.
- **N2, confermata esplicitamente:** K quantifica composizionalmente i risultati ternari individuali dei candidati, secondo S/F/U e l’inviluppo [S,S+U], senza inferenza globale sulle dipendenze fra candidati distinti. La fixture dei due ordini e del rimborso con chiave ignota resta UNKNOWN anche se una prova globale potrebbe dimostrare almeno un ordine senza rimborso. Restano tutte le dipendenze interne obbligatorie di E, P, share, guardie e ownership. Non è un’opzione degli executor.
- **N3:** gli ignoti richiedono UNKNOWN se possono cambiare la verità del test completo richiesto, non per la sola identità incerta del vincitore. La fixture Scienza=0, Arte=10, Tecnologia=10 e una categoria ignota dà NO MATCH a “Scienza unica prevalente”; unicità e massa ignota restano preservate.

### N.2 Perimetro e verifiche del consolidamento

Consolidati esclusivamente i quattro documenti 06/07/08/13 v1.2, rimuovendo il suffisso `-proposta`, aggiornando stato e collegamenti e distinguendo i richiami storici dagli obblighi vigenti. Nessuna modifica alle decisioni semantiche approvate.

- Rigenerato 08 dalla sezione finale del 07 definitivo e verificata l’identità byte per byte.
- Confrontati i 94 blocchi di codice del 07 con la proposta approvata: costruttori, payload e binding invariati. Preservati V/E/K/P/T, limiti strutturali, 26 operatori Record, core.Filter e relative regole.
- Confrontate letteralmente le fixture 06 §§28.19–28.20 con la versione 1.1 e con la proposta approvata: testo, JSON, tabelle ed esiti identici. NativeCurrency ignota e JSON raw Null rimangono distinti.
- Controllati N1/N2/N3, fixture e rinvii, collegamenti locali, stato normativo attuale e assenza di diciture che presentino la 1.2 come non approvata. Le sezioni K/L/M rimangono registrazioni storiche delle revisioni, non una richiesta di nuova approvazione.
- Confrontati gli SHA-256 di tutti gli altri documenti preesistenti in filtering: versioni 1.1, benchmark 03, audit e rapporti 11/12 invariati. Verificati `git diff --check`, `git diff --cached --check` e il perimetro dei quattro file nel commit documentale dedicato `Approve unified filtering specifications v1.2`.
- Nessuna modifica applicativa, ai driver o agli altri file estranei; nessun test runtime o accesso a warehouse necessario per questo consolidamento. I controlli sono documentali e non attestano un’implementazione della Fase 4B.

### N.3 Rinvii infrastrutturali e architettura separata

Restano **rinviati e non risolti** Identity Resolution/pubblicazione Profiles, acquisizione asincrona/attribuzione KPID e matching delle destination. Le rispettive assunzioni non attestano garanzie già implementate e possono condizionare il rilascio senza bloccare la progettazione del compilatore.

Il rapporto 12 rimane una proposta architetturale non approvata e non modificata. La successiva revisione della Fase 4B valuterà la direzione preferita di un compilatore semantico condiviso con moduli di interrogazione tipizzati e composizione SQL comune, verificando la sufficienza di un contratto fisico più piccolo. Piano fisico concreto, API warehouse, materializzazione e strategie listing/count/export restano da decidere separatamente. Nessuna alternativa del 12 è adottata automaticamente e la Fase 4B non è implementata da questo consolidamento.
