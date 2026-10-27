# Behavioral Filtering Product & Semantic Specification

## Specifica semantica riconciliata — Versione 1.1 (proposta da approvare)

## 1. Status and scope

**Stato:** **proposta di revisione normativa 1.1, 26 ottobre 2026; NON ancora approvata**. Sostituisce la versione congelata 1.0 soltanto dopo approvazione. **Prodotto:** record filtering e unified profile filtering di Krenalis, con un unico contratto pubblico `core.Filter`. La semantica delle nuove capacità non ne attesta l’implementazione o la disponibilità dei dati.

### 1.1 Autorità e base documentale

La specifica congelata 1.0 del 6 settembre 2026 ha sostituito la Fase 3. Questa **proposta 1.1** riconcilia quei requisiti con il contratto implementato del record filtering: il vecchio congelamento non prevale automaticamente sulle decisioni di compatibilità ora assunte. La nuova revisione resta proposta finché non viene approvata; eventuali contraddizioni residue sono evidenziate, non risolte implicitamente.

Nella **consolidazione storica della versione 1.0** furono utilizzati i tre allegati allora effettivamente disponibili: `Pasted text(20260906-155008).txt`, contenente la richiesta e le decisioni finali; `behavioral_filtering_fase3_specifica_prodotto_semantica(2).md`, contenente la specifica originale; `intent_out_of_sample_fissati(1).md`, contenente venti intenti sintetici fissati per la review. Quest'ultimo non contiene la matrice o le conclusioni complete dell'Adversarial Review: nessuna ulteriore conclusione del red-team viene attribuita a un testo non fornito.

Gli identificativi I001–I100 e S001–S015 conservano il riferimento al benchmark riportato dalla Fase 3; gli identificativi O01–O20 appartengono all'allegato out-of-sample. Le classificazioni CORE/COMMON/ADVANCED/RARE non vengono riassegnate. Non viene svolto un nuovo benchmark, né viene attribuita una lettura indipendente a documenti di Fase 1 o Fase 2 non allegati a questo consolidamento.

### 1.2 Convenzioni e confine normativo

**MUST** indica un requisito vincolante nella specifica proposta; **MUST NOT** un comportamento vietato; **SHOULD** e **SHOULD NOT** indicano raccomandazioni derogabili soltanto con motivazione; **MAY** indica una possibilità compatibile, non automaticamente compresa nella revisione 1.1. Questi requisiti diventano normativi alla sua approvazione.

Le sezioni 1–30 sono normative per definizioni, tabelle semantiche, limiti, prerequisiti e risultati attesi. Le frasi contrassegnate **Esempio informativo**, le altre esemplificazioni business e i rationale descrittivi illustrano esclusivamente forme già definite e non introducono capability. Le definizioni matematiche e le tabelle di confronto semantico rimangono normative anche quando accompagnate da esempi. Nella sezione 28 le fixture sono esempi di progetto; gli esiti attesi, condizionati ai prerequisiti dichiarati, sono criteri normativi di conformità e non risultati di un'esecuzione. I paragrafi **Rationale non normativo** e il changelog documentano le ragioni e le modifiche, senza costituire una seconda specifica.

“Baseline” indica le forme semantiche di questa versione, non un obbligo di disponibilità universale di ogni fonte o adapter. “Inline” indica una costruzione esprimibile dall'utente mediante il linguaggio chiuso, indipendentemente da come viene eseguita. “Condizionale” indica una capability consumabile soltanto quando esiste il contratto o il producer richiesto.

Per le capacità e le condizioni qui definite, il documento stabilisce il significato normativo necessario a progettare AST, API e UI; le questioni **tecniche ancora aperte**, elencate nel changelog di riconciliazione, non sono risolte per implicazione. Non definisce JSON wire, database, piano di esecuzione o algoritmi di dominio.

---

### 1.3 Raccordo normativo con `core.Filter`

`core.Filter` è il **contratto pubblico comune**. Il **record filtering** verifica condizioni sulle proprietà di un singolo record (evento, utente importato o profilo) mediante operatori e gruppi AND/OR già disponibili. Lo **unified profile filtering** include tali funzionalità e aggiunge le cinque primitive comportamentali V/E/K/P/T nel perimetro dei profili unificati. Il nome *behavioral filtering* designa il progetto, non un secondo linguaggio pubblico indipendente.

**Baseline implementativa per le condizioni condivise:** branch `typed-filter-values`, commit `d69e61681b2da531bbd9fdd067c825b589338f97`, secondo il resoconto di Codex e la revisione indipendente. Il commit `e66eea846` resta la baseline storica dell'audit 10 e del rapporto di riconciliazione precedente. Non si attribuisce qui un'ulteriore verifica del repository. Snowflake reale non è stato verificato per il commit nuovo.

La compatibilità si riferisce alla struttura pubblica `Filter {operator, rules}`, alle condizioni `FilterCondition {property, operator, values}`, ai 26 operatori, alle combinazioni operatore/tipo, alla validazione e ai risultati semantici stabiliti dopo `d69e6168`. Una nuova condizione non può reinterpretare i literal o i path delle condizioni già salvate. La correzione della semantica dei missing in `d69e6168` è **un cambiamento intenzionale già effettuato**, non una migrazione implicita derivante dalla presente revisione.

L'estensione Unified è ammessa **solo** nelle query dei profili unificati tramite API e nella pagina Profiles dell'Admin e nelle destination connections che esportano **profili** verso applicazioni, file e database. Negli altri contesti il record filtering rimane l'unico livello disponibile, ove già ammesso; un contesto che oggi vieta i filtri non viene abilitato da questa revisione. Il server MUST respingere esplicitamente le forme non autorizzate, non ignorarle o degradarle in silenzio.

La revisione 1.1 usa inoltre l’audit 10, il rapporto di riconciliazione, la verifica Codex della baseline `d69e6168` e il registro sintetico del 26 ottobre 2026. Il documento 03 resta il benchmark indipendente dei requisiti business; il documento 07 revisionato progetta il modello canonico tecnico; il documento 08 ne è soltanto un estratto. L'audit 10 resta storico e non viene riscritto retroattivamente.

## 2. Executive product decision

Nella modalità **unified profile filtering** il prodotto MUST rimanere un filtro di profili, non un editor di query. L’utente descrive fatti correnti, attività, esperienze, esiti e abitudini; il risultato determina membership all’istante di valutazione T. Il **record filtering** del medesimo `core.Filter` continua invece a valutare il singolo record nel suo contesto operativo e non acquisisce per questo storia comportamentale o un T storico.

La semantica usa esattamente cinque primitive: **Confronto tipizzato**, **Esperienza selezionata**, **Insieme per identità**, **Percorso ordinato**, **Schema di attività**. Queste forme non prescrivono cinque categorie visibili, cinque card o cinque tipi di rappresentazione tecnica.

Il modello MUST preservare la medesima occurrence, la selezione prima della verifica, l'identità pertinente, l'ordine quando richiesto, le finestre e le deadline, la grana e l'adeguatezza dell'evidenza. Contesti lessicali, corpi chiusi, ruoli tipizzati, slot autorizzati e limiti strutturali operano congiuntamente: la sola località non è sufficiente a impedire un query language.

Somma, media, count, distinti e gli altri operandi atomici elencati in §10 sono inline. Confronti, difference, ratio, relative change e share sono forme chiuse; l'aritmetica terminale non produce operandi concatenabili. Le costruzioni contestuali non esportano testimoni.

Lo stato autorevole è consumato tramite contratti mirati. I loro named evidence scopes sono finiti, tipizzati e correlati automaticamente all'oggetto proprietario. Le evidenze derivate certificate entrano attraverso le normali primitive, senza importare nel filtro gli algoritmi che le producono.

Il percorso ammette al massimo una chiave semantica e un solo blocco contiguo di correlazione; i passi esterni non ereditano la chiave. Lo schema di attività rimane un catalogo chiuso e comprende la cardinalità dei periodi qualificati.

Soltanto MATCH determina appartenenza. NO MATCH e UNKNOWN escludono il profilo per ragioni distinte; una precedente appartenenza non sopravvive a un risultato corrente UNKNOWN. La copertura del core, quella dipendente da contratti autorevoli e quella dipendente da derived evidence MUST essere riportate separatamente secondo §26. I051 non comporta un producer video obbligatorio nel core.

---

## 3. Normative principles

**Fedeltà alla domanda.** Il prodotto MUST conservare tutte le distinzioni che possono cambiare membership: insieme osservato, grana, unità, periodo, selezione, oggetto comune, riferimento temporale, deadline, non-vuoto e adeguatezza dell'evidenza. Un'etichetta business MUST NOT cancellarle dal filtro salvato.

**Una valutazione coerente.** Tutte le condizioni MUST usare lo stesso istante di valutazione, indicato qui con **T**, e una vista coerente dei dati correnti e delle evidenze. Il tempo degli eventi è quello dell'azione, non quello di ricezione. Il filtro non attribuisce causalità a una successione temporale.

**Identità prima dei conteggi.** Contano azioni business distinguibili, non copie tecniche. Contare ordini, righe, unità, prodotti e operazioni di rimborso sono richieste differenti. Il prodotto MUST rendere visibile la grana scelta.

**Nessuna approssimazione invisibile.** Il prodotto MUST NOT sostituire “mai” con “negli ultimi 90 giorni”, “ultimo” con “ultimo noto che soddisfa il filtro”, “stesso prodotto” con “qualunque prodotto”, oppure stato passato con stato corrente.

**Potenza senza programmabilità generale.** La possibilità di calcolare un risultato non basta a renderlo esprimibile. Una nuova capacità MUST entrare come operazione tipizzata, estensione di un contratto circoscritto o nuova proposta di prodotto; non come formula libera o riferimento arbitrario.

**Evidenza ispezionabile.** Una spiegazione MUST identificare la condizione, i suoi parametri e l'evidenza decisiva: un fatto, un conteggio con grana, un'esperienza, un oggetto, un percorso o un intervallo. Per un'assenza, la spiegazione MUST mostrare anche che cosa e fino a quando è stato osservato. Non occorre mostrare ogni possibile match.

**Separazione fra linguaggio e disponibilità.** Capacità esprimibile, mapping configurato, dati sufficienti ed esecuzione riuscita sono quattro aspetti diversi. Un requisito dati mancante MUST NOT essere presentato come “il profilo non soddisfa la condizione”.

**Tre livelli separati.** Lo **user language** presenta la domanda business; il **semantic filter model** ne determina significato, contesti e risultati; l'**execution model** la realizza. La UI MAY cambiare organizzazione e il motore MAY cambiare strategia, ma entrambi MUST conservare la semantica. Il costo, la precomputazione o la statefulness non cambiano da soli il confine inline/derived.

**Decisioni numeriche esatte.** Le soglie MUST essere applicate ai valori nella precisione e nell'unità dichiarate dal contratto, non ai numeri arrotondati per la visualizzazione. MUST NOT essere introdotti epsilon, arrotondamenti decisionali o tolleranze implicite.

**Nessuna dipendenza nascosta.** Una condizione sorella MUST NOT leggere un'evidenza locale creata da un'altra condizione. Il nome di una recipe, l'etichetta di un evento o la pubblicazione di uno scalare MUST NOT creare tale dipendenza.

---

## 4. User mental model

Il modello mentale è: **«Descrivo persone che oggi si trovano in una certa situazione, sulla base di ciò che sappiamo e di ciò che hanno fatto».**

| Domanda dell'utente | Forma riconoscibile | Cosa resta esplicito |
|---|---|---|
| Che cosa sappiamo oggi? | “Vive in Italia”; “ha un ticket aperto” | Stato corrente e fonte pertinente |
| Che cosa ha fatto, e quanto? | “Ha acquistato”; “ha speso €1.000”; “ha usato cinque funzionalità” | Attività, periodo, quantità e unità |
| Quale esperienza conta? | “Il suo primo acquisto”; “dopo l'ultimo upgrade” | Esperienza scelta e controlli su quella esperienza |
| Come è andata una situazione? | “Ha visto e poi acquistato lo stesso prodotto”; “non ha risposto a quel messaggio” | Oggetto, ordine, occasione e orizzonte dell'esito |
| Quale abitudine o cambiamento riconosciamo? | “Ogni settimana”; “è tornato dopo una pausa”; “usa la metà di prima” | Calendario, confronto personale e pausa effettiva |

L'utente non deve imparare il termine “contesto lessicale”. Deve vedere frasi come **“per questo ordine”**, **“nei giorni successivi a quell'upgrade”** e **“fra i prodotti di quel carrello”**. Quando due condizioni sono indipendenti, il prodotto MUST NOT suggerire che si riferiscano allo stesso episodio.

Le condizioni si aggiungono normalmente al profilo. Una richiesta con “quel”, “lo stesso” o “da allora” apre invece una costruzione guidata che mantiene insieme i requisiti pertinenti. La spiegazione gerarchica mostra questo legame senza introdurre variabili.

---

## 5. User-facing condition strategy

Il catalogo presenta quattro ingressi orientativi, attraversati dalla ricerca. Non replica la tassonomia tecnica.

| Ingresso | Richieste generiche principali | Accessi business alla stessa capacità |
|---|---|---|
| **Profilo e situazione attuale** | Proprietà; dato mancante/null; confronto fra proprietà; situazione corrente supportata | Contattabile; abbonato pagante; ticket aperti; trial corrente |
| **Attività e valore** | Ha fatto/non ha fatto; quante volte; totale/media; valori diversi; quota; valore più frequente | Ha speso; valore medio ordine; quantità acquistate; interesse prevalente |
| **Esperienze ed esiti** | Prima/ultima esperienza; prima e dopo una tappa; stesso oggetto; percorso; esito entro una scadenza | Primo acquisto; riacquisto; carrello non completato; mancata risposta; ordine rimborsato |
| **Abitudini e cambiamenti** | Giorni/settimane attivi; intensità concentrata; continuità; confronto fra periodi; ritorno | Usa ogni settimana; ha smesso; è tornato; utilizzo in calo |

Ogni ingresso SHOULD mostrare poche famiglie riconoscibili, non tutte le combinazioni di parametro. La ricerca MUST trovare anche la variante specialistica senza obbligare a navigare tutte le categorie.

**Generic condition.** Espone una capacità riutilizzabile su comportamenti o proprietà selezionabili: “numero di azioni”, “ultimo evento”, “stesso valore ripetuto”.

**Business recipe.** Espone una frase di dominio costruita sulle stesse capacità. “Ha speso” imposta il ruolo Purchase, l'importo monetario e la grana ordine; l'utente sceglie almeno periodo, lordo/netto, valuta di confronto e soglia. Non deve cercare `Purchase → amount → SUM`.

**Configurazione specialistica.** Amplia una capacità già riconosciuta: confronto con un'altra misura, base del primo/ultimo, criterio di opportunità, stessa identità o dimensione della ripetizione. Questi controlli non sono prerequisiti per trovare somme, medie e distinti.

Una ricerca senza il mapping necessario MUST mostrare la capacità e il requisito mancante, distinguendola da un filtro attivabile. MUST NOT creare una falsa equivalenza fra eventi raw che hanno soltanto nomi simili.

---

## 6. Semantic model overview

Le cinque primitive sono **forme di significato**, non decisioni su AST, serializzazione o card. Le sigle seguenti sono soltanto notazione didattica.

| Primitiva | Significato e input concettuali | Output e scope | Configurabile | Deliberatamente non configurabile | Intenti e recipe; motivo della distinzione |
|---|---|---|---|---|---|
| **V — Confronto tipizzato** | Verifica un fatto corrente, una misura su uno scope, oppure un confronto chiuso fra due valori compatibili | MATCH/NO MATCH/UNKNOWN sul profilo; nessun riferimento a occurrence esportato | Proprietà, scope, riduzione ammessa, operatore, soglia, guardia minima, secondo valore ammesso | Formule ricorsive, funzioni arbitrarie, riferimenti a testimoni di altre condizioni | I001–I024, I030, I038–I040, I045, I047, I049, I054, I056, I063, I087, I095, I100. Alimenta spesa, frequenza, quota, preferenza e calo. Conclude test su valori senza restituire un'esperienza da proseguire |
| **E — Esperienza selezionata** | Sceglie una occurrence coerente, o un blocco limitato di ultime occurrence, da una base esplicita; poi verifica requisiti | Risultato sul profilo; occurrence e ancora soltanto locali | Primo/ultimo/N-esimo, base, verifiche successive, finestre relative, preset di confronto degli ultimi valori | Ordinamento per importo, selettori arbitrariamente annidati, confronto fra posizioni arbitrarie, esportazione dell'ancora | I025–I037, I068, I093, I096; S014. Alimenta prima conversione, ultimo NPS, prima/dopo upgrade. È distinta perché la scelta deve precedere le verifiche |
| **K — Insieme per identità** | Valuta requisiti su ciascuna identità di un insieme definito: valore osservato, oggetto correlato, opportunità o situazione autorevole supportata | Risultato sul profilo mediante quantificatore; chiave e pivot soltanto locali | Fonte dell'insieme, ruolo della chiave, requisiti locali ammessi, alcuni/tutti/nessuno/conteggio/quota, selezione limitata di opportunità | Join su condizioni arbitrarie, più dimensioni liberamente raggruppate, relazioni navigabili, creazione ricorsiva di insiemi | I009, I041, I043, I048, I050, I053, I059, I061, I072–I078, I099; S002, S004, S009. Alimenta riacquisto, risposta, rimborso e situazione aperta. È distinta perché i requisiti devono rimanere sulla stessa identità, anche senza ordine |
| **P — Percorso ordinato** | Cerca una tupla finita di occurrence con ordine, eventuale blocco contiguo correlato e vincoli temporali/esiti circoscritti | Risultato sul profilo; passi, chiave e intervalli soltanto locali | Da 2 a 5 passi; basi dei passi; precedenza; ritardi; una identità comune a tutti i passi o a un solo blocco contiguo; esclusioni fra estremi; esito dal passo finale | Percorsi ricorsivi, branching di processi, espressioni generali fra proprietà dei passi, conteggio delle combinazioni di match | I065–I071, I079; S012. Alimenta conversione, onboarding ordinato e percorso senza frizioni. È distinta perché deve scegliere congiuntamente più esperienze coerenti |
| **T — Schema di attività** | Riconosce uno dei pattern temporali enumerati su un comportamento: calendario, cardinalità di periodi qualificati, burst, continuità o ritorno | Risultato sul profilo; solo il ritorno selezionato può aprire il corpo locale autorizzato | Pattern, scope, periodo, calendario, soglie, durata di pausa; conteggio o ultimo ritorno | Espressioni time-series, finestre generate da formule, statistiche arbitrarie, episodi riutilizzabili fra condizioni | I080–I086, I090–I092; S011. Alimenta abitudine, attività concentrata, continuità e riattivazione. È distinta perché calendario, pause e intervalli mobili non sono semplici conteggi dello stesso insieme |

**Responsabilità chiuse.** V confronta proprietà e scalari, usa le riduzioni atomiche e conclude test terminali esplicitamente ammessi: MUST NOT diventare un expression tree. E prima seleziona e poi verifica. K parte da candidati identificabili, applica il corpo autorizzato per la forma e conclude tramite un quantificatore; grouping per dimensione, correlazione di eventi e stato autorevole non sono la stessa algebra. P ricerca tuple finite ordinate, non arbitrary path expressions. T è un catalogo chiuso di temporal activity recognizers, non un algoritmo temporale universale.

Esistenza e frequenza sono usi di count. Distinti, quote e i test chiusi di prevalenza appartengono a V; ripetizione per valore e novità appartengono a K. Recipe, Authoritative Object Contracts e Certified Derived Evidence non sono primitive aggiuntive. Gli intenti nella tabella identificano responsabilità semantiche, non dichiarano disponibilità incondizionata dei relativi dati o producer.

Tutte le primitive, quando compaiono nel filtro principale, producono un risultato sul profilo. Le misure e i testimoni interni non diventano variabili. Una stessa implementazione futura potrà condividere operazioni fra più primitive senza cambiarne il significato.

**Validità dei parametri.** I conteggi e le soglie cardinali sono interi non negativi; i minimi di campione, “almeno un candidato”, nth, ultime N e le lunghezze positive di un pattern usano interi almeno pari a uno. Le durate configurate sono positive. Un intervallo assoluto immesso deve avere inizio precedente alla fine; un intervallo dinamico “ciclo corrente finora” può essere vuoto nell'istante iniziale senza diventare un dato invalido. Le soglie di una share sono fra 0% e 100%; le ratio non sono percentuali di un sottoinsieme e non hanno tale limite implicito. Gli operatori devono essere applicabili al tipo e all'unità. Vincoli staticamente incompatibili o un superamento dei limiti strutturali rendono invalida la configurazione; valori problematici soltanto per alcuni profili seguono le regole di evidenza di §19.

---

## 7. Local context model

### 7.1 Regola lessicale normativa

Le costruzioni contestuali possono creare evidenze locali, ma tali evidenze MUST essere **lessicalmente locali, tipizzate e consumabili soltanto dagli slot esplicitamente autorizzati dalla costruzione che le possiede**.

Un riferimento a occurrence, chiave, oggetto, opportunità, passo o ritorno MUST nascere e terminare nella stessa costruzione. Una condizione sorella MUST NOT leggerlo. Il riferimento MUST NOT essere esportato, nominato dall'utente come variabile o usato come input arbitrario di altre espressioni. Una recipe MUST NOT renderlo esportabile. Pubblicare uno scalare derivato MUST NOT rendere nuovamente accessibile il testimone di origine.

Due condizioni sorelle hanno testimoni indipendenti anche quando usano la stessa etichetta o recipe. Copiare una condizione non condivide il testimone. Un gruppo AND non introduce correlazione. Il profilo corrente e T non sono local bindings: possono essere letti dove la specifica li ammette.

La protezione del perimetro MUST basarsi sulla combinazione di località, corpi chiusi, ruoli tipizzati, slot autorizzati e limiti strutturali. MUST NOT essere introdotti variabili, join o subquery general purpose, neppure entro un singolo contesto locale.

### 7.2 Corpi ammessi per forma

| Costruzione proprietaria | Evidenza locale | Slot autorizzati | Confine |
|---|---|---|---|
| Esperienza singola selezionata | Una occurrence e il suo momento | Predicati sulla occurrence; fino a quattro scope comportamentali foglia prima/dopo il momento; confronti tipizzati; un eventuale slot “primo seguito” | Nessun E, K, P o T liberamente annidato |
| Ultime N esperienze | Blocco ordinato di N occurrence | Lo stesso predicato su alcune/tutte/un numero delle occurrence; preset ultimo-vs-precedente o ultimi-due-intervalli | Nessun corpo arbitrario per ogni posizione |
| Valore osservato | Una dimensione del repertorio, per candidato | Misure foglia del medesimo repertorio sulla stessa dimensione, anche in periodi diversi; ripetizione e novità | Non correla repertori arbitrari tramite coincidenza di valori |
| Oggetto correlato negli eventi | Una identità semantica per candidato | Fino a quattro scope foglia automaticamente ristretti alla stessa identità; pivot primo/ultimo della sorgente primaria; proprietà e misure autorizzate | Nessun cambio di chiave o navigazione verso altri oggetti |
| Opportunità | Identità distinta e occurrence iniziale | Proprietà dell'opportunità; esito attribuito alla medesima opportunità; orizzonte/deadline; quantificazione degli esiti | L'esito non apre una nuova opportunità |
| Oggetto autorevole | Candidato dell'inventario corrente certificato | Proprietà, identity e lifecycle ammessi; named evidence scopes dichiarati dal contratto; esclusivamente gli eventuali slot temporali o membri già enumerati in §15 | Nessun body universale, comportamento arbitrario o RelationCondition |
| Percorso | Da 2 a 5 occurrence; eventualmente identità comune nel solo blocco correlato | Predicati sui passi; ordine, durata adiacente/totale; assenza fra estremi ammessi; esito dal finale | Nessun sotto-percorso; nessun accesso alla chiave fuori dal suo blocco; nessuno slot verticale di dominio |
| Ultimo ritorno qualificante | Ritorno canonico con uso precedente e pausa verificata | Requisiti sul ritorno e corpo foglia dell'esperienza singola | Nessuna data esportata a una condizione sorella |

L'unione degli slot delle righe MUST NOT diventare il corpo disponibile a ciascuna riga. L'ammissione di un nuovo slot in un contratto non autorizza quello slot nei contratti o nelle forme sorelle.

Uno **scope foglia** può filtrare occurrence, contarle o applicare le misure ammesse; MUST NOT contenere un filtro di profili o un costruttore contestuale. Nei corpi che ammettono attività subordinate il limite è quattro scope foglia per costruzione, comprendendo quelli raggiunti tramite l'eventuale membro autorizzato. Il limite non conta i predicati scalari né i passi propri di P. Due operandi che richiedono due scope subordinati consumano due posti; una guardia sul medesimo scope non crea una nuova fonte. Il sottoinsieme interno di una share non crea un contesto navigabile aggiuntivo.

Lo slot **“primo seguito”** consente soltanto di selezionare la prima occurrence di un comportamento successiva all'ancora, fino a T, e verificarne proprietà o ritardo. MUST NOT creare nuove finestre ancorate al seguito. È una forma chiusa, non selezione ricorsiva.

### 7.3 Un solo passaggio a membri autorizzato

È ammessa la costruzione **carrello corrente → uno dei prodotti effettivamente contenuti in quel carrello → attività del profilo sullo stesso prodotto**. Il prodotto testimone MUST appartenere al carrello che soddisfa gli altri requisiti; non può aprire relazioni verso categorie, fornitori, altri carrelli o altri profili.

La struttura massima è un frame principale, un eventuale frame membro autorizzato, poi scope foglia. Un limite di profondità non autorizza abbinamenti ulteriori. Una verifica su una riga posseduta da un ordine rimane accesso a un membro della stessa evidenza (§9), non navigazione fra oggetti arbitrari.

### 7.4 Connessioni tipizzate, non composizione universale

Il ritorno canonico può alimentare soltanto l'esperienza “ultimo ritorno qualificante” nella forma qui ammessa. Una certified occurrence o milestone può alimentare una normale base di selezione o un passo soltanto se il suo contratto certifica i ruoli richiesti (§16). Ciò non permette di collegare qualunque output a qualunque input.

Un timestamp estratto da un'esperienza MUST NOT essere registrato come generica “data personale” per aggirare la località. Un dato esterno pubblicato e certificato ha il proprio contratto e può essere consumato come tale; non restituisce automaticamente l'occurrence, l'oggetto o gli altri attributi del testimone originale. Una semplice pubblicazione scalare non è una certificazione di occurrence.

### 7.5 Limiti strutturali

| Struttura | Limite della proposta 1.1 |
|---|---|
| Gruppi AND/OR nel filtro principale | **Quattro livelli di gruppi, radice inclusa** |
| Nodi discendenti nell’albero principale | **100**, gruppi e condizioni compresi; radice esclusa |
| AND/OR in un corpo contestuale | Due livelli, soltanto fra foglie autorizzate |
| Scope comportamentali subordinati | Quattro per costruzione che li ammette |
| Esperienze nth / ultime N / ultime N opportunità | N da 1 a 10 |
| Passi positivi di un percorso | Da 2 a 5, occurrence distinte |
| Correlazione di un percorso | Al massimo una chiave, in un solo blocco contiguo di almeno due passi |
| Passaggio a membri | Soltanto il percorso carrello-prodotto autorizzato |

Nella forma pubblica `core.Filter` il filtro assente non aggiunge vincoli; la radice di un filtro presente è un gruppo AND/OR; i gruppi vuoti sono invalidi, mentre i singleton e i gruppi consecutivi con lo stesso operatore sono ammessi. Una nuova `FilterRule` occupa **un nodo discendente** nell’albero principale; i suoi componenti interni rispettano inoltre i budget propri di V/E/K/P/T, senza moltiplicare il conteggio principale o aggirare i limiti dei body. L’ammissibilità MUST essere controllata **prima** di eventuali semplificazioni canoniche; la forma pubblica non viene riscritta automaticamente. Il dettaglio della rappresentazione dei corpi resta tecnico per la Fase 4B.

Aumentare un limite MAY costituire un’evoluzione compatibile solo secondo §29; MUST NOT autorizzare automaticamente nuove combinazioni di ruoli, slot o contesti.

---

## 8. Profile/current-state semantics

Le proprietà di profilo rappresentano lo **stato corrente a T**. Sono ammesse proprietà scalari annidate, insiemi di valori dichiarati e confronti fra due proprietà correnti compatibili. Il percorso verso un campo annidato non è una relazione verso un altro soggetto.

| Tipo di richiesta | Semantica normativa |
|---|---|
| Uguaglianza/diversità | Valori confrontabili dello stesso dominio; nel JSON dinamico, un valore presente di kind differente è distintamente non uguale (senza coercizione). Chiave assente: UNKNOWN per entrambi |
| Confronto ordinato/intervallo | Numeri, quantità, tipi temporali e **testo ordinabile già ammesso dal record filtering**; estremi delle forme record inclusi, salvo operatori stretti. Le finestre comportamentali mantengono i bordi specifici di §17 |
| Appartenenza | Valore in un insieme; per una proprietà multivalore, “contiene” è esistenziale sugli elementi dello stesso insieme |
| Presenza e qualità del valore | Proprietà ordinaria assente; percorso JSON assente; JSON `null` esplicito; stringa e contenitori vuoti sono trattati dagli operatori esistenti secondo tipo e stato. La proprietà ordinaria dello schema Profiles non è nullable |
| Confronto fra proprietà | Due valori dello stesso profilo corrente; unità e domini compatibili; nessuna formula libera |
| Testo | Uguaglianza, diversità, appartenenza, contiene/inizia/finisce con testo letterale e **ordinamento** per il sottoinsieme di String già ammesso. Nessuna regex o codice |

Il confronto di identificatori è esatto sul dominio semantico dichiarato. Normalizzazioni di paese, lingua, maiuscole o Unicode MUST essere dichiarate dal catalogo e applicate coerentemente: il builder non “corregge” silenziosamente valori. Per il testo libero il confronto è esatto salvo normalizzazione esplicitamente indicata.

Le quattro fonti di significato temporale — stato corrente del profilo, proprietà acquisita nell'evento, stato storico certificato e stato corrente autorevole dell'oggetto — MUST rimanere distinte secondo §18. I normali confronti evento–profilo corrente sono ammessi quando esplicitamente etichettati “attuale”.

Per esempio, I010 confronta due lingue correnti; I063 confronta il paese di un utilizzo con il paese corrente. Né l'uno né l'altro richiedono di ricostruire lo stato passato.

Contattabilità e abbonamento attivo non sono sinonimi di email presente e avvio storico di una subscription. Il primo usa una recipe di policy corrente esplicita; il secondo una situazione autorevole del dominio. Questa specifica non definisce una policy legale di consenso.

---

### 8.1 Catalogo condiviso dei 26 operatori del record filtering

Gli operatori seguenti conservano i nomi pubblici e le regole di compatibilità dei tipi del commit `d69e6168`. **La disponibilità nel catalogo non implica che ogni tipo o slot supporti ogni operatore.** Questa sezione specifica il repertorio condiviso, non introduce un nuovo formato wire. Le denominazioni seguenti sono quelle pubbliche esistenti.

| Operatori | Tipi / semantica essenziale | Operandi |
|---|---|---|
| `is`, `is not` | Scalari ammessi dal record filtering, eccetto Boolean ordinario; JSON String/Number con uguaglianza tipizzata | 1 |
| `is less than`, `is less than or equal to`, `is greater than`, `is greater than or equal to` | Numeri, String senza enum, JSON String/Number; ordine nel tipo effettivo | 1 |
| `is between`, `is not between` | Numeri, String senza enum, DateTime/Date/Time/Year, JSON due String o due Number dello stesso kind; estremi inclusi | 2 |
| `contains`, `does not contain` | String (sottostringa), Array typed ammessi (elemento immediato), JSON String/Array dinamico con elemento String/Number | 1 |
| `is one of`, `is not one of` | Scalari ammessi; JSON lista mista String/Number/Boolean/Null, senza coercizione | ≥1 |
| `starts with`, `ends with` | String senza enum; JSON String | 1 |
| `is before`, `is on or before`, `is after`, `is on or after` | DateTime, Date, Time, Year; non applicabili a JSON | 1 |
| `is true`, `is false` | Boolean ordinario oppure JSON Boolean | 0 |
| `is empty`, `is not empty` | String, JSON, Array, Map, con limitazioni dell'enum; Object di schema non ammesso | 0 |
| `is null`, `is not null` | Proprietà Nullable nei contesti che le ammettono, oppure JSON; Profiles ordinario non è Nullable | 0 |
| `exists`, `does not exist` | Proprietà ReadOptional oppure percorso interno JSON; Object di schema conserva le restrizioni della rappresentazione | 0 |

Gli operandi conservano il proprio kind JSON (`100`, `"100"`, `true`, `null` sono diversi), la cardinalità, la validazione e la precisione della baseline: Int/UInt64, Float alla precisione del tipo, Decimal e Number JSON entro 76 cifre di precisione e 37 di scala, Time/Year e tipi temporali, UUID/IP normalizzati, tipi con enum, semantiche già dichiarate di denaro/unità/durata. Gli otto operatori unari richiedono `Values == nil` (omissione o `values:null`); `values:[]` non è un unario valido. Un elemento `[null]` è invece un operando JSON Null ammesso nella membership JSON, non un valore assente. I path rimangono quelli pubblici già supportati; non diventano JSONPath né riferimenti a un'altra proprietà. In particolare `values: ["otherProperty"]` rimane un literal String, non un riferimento. Il confronto fra proprietà è una **nuova** forma esplicita.

### 8.2 Valore assente, JSON `null` e tipi non applicabili

Ai **20 operatori sui valori** (i primi otto gruppi della tabella, inclusi `is true` e `is false`) la mancanza della proprietà ordinaria o del percorso JSON assegna **SQL `NULL`** e, nell'applicabilità Go `Applies`, `false`, anche per gli operatori negativi e per `nil` Go ordinario. SQL `NULL` è il risultato ternario di un confronto non determinabile, non un nuovo valore immesso nel record. Il `WHERE` ordinario seleziona solo TRUE.

Il JSON `null` è un **valore JSON esplicitamente presente** e non equivale a un path mancante. In uguaglianza e diversità JSON, un valore presente di kind differente produce rispettivamente FALSE e TRUE: JSON `null` e String `"100"` sono diversi da Number `100`. La membership JSON mantiene l'eventuale letterale `null` dentro la lista. Per ordinamento, intervallo, contenimento, prefisso e suffisso, un kind JSON su cui l'operazione non è applicabile produce SQL `NULL` / Go `false`, anche nella variante negativa; non si converte implicitamente String `"200"` nel Number `200`. `is true` / `is false` verificano il Boolean effettivo e, per JSON presente non Boolean, conservano il FALSE della baseline.

I **sei operatori sullo stato del dato** (`is empty`, `is not empty`, `is null`, `is not null`, `exists`, `does not exist`) mantengono **le regole specifiche della baseline**, senza trasformare automaticamente missing in UNKNOWN. Per esempio, su path JSON certamente assente: `exists` = FALSE, `does not exist` = TRUE, `is null` = FALSE, `is not null` = TRUE, `is empty` = TRUE e `is not empty` = FALSE. Su JSON `null` presente: `exists` = TRUE e `is null` = TRUE. `is empty` riguarda anche contenitori e assenza, non soltanto la String vuota; `is not null` non equivale universalmente a “valore noto”.

Questa prescrizione **non risolve in modo implicito** le divergenze storiche fra una proprietà ordinaria nullable rappresentata in Go e una colonna SQL `NULL`, o la presenza di Object di schema appiattiti: tali casi mantengono lo status di questioni tecniche aperte e non sono attestati equivalenti. Una configurazione invalida e un fallimento tecnico restano errori, non SQL `NULL` semantici. Un dato JSON lecito di tipo non applicabile è invece differente da un filtro malformato.

## 9. Behavioral/event scope

### 9.1 Definizione

Adottiamo un **Behavior Scope** riutilizzabile come concetto, non come query salvabile con parametri liberi:

> Un repertorio di comportamenti, alla grana dichiarata, con requisiti sulla medesima occurrence, in un periodo esplicito, eventualmente ristretto dalla chiave o dall'ancora del contesto proprietario.

Lo scope identifica occurrence certamente qualificanti, occurrence la cui qualificazione è ignota, ed eventuali lacune di osservazione. MUST NOT trasformare automaticamente le occurrence incerte in occurrence scartate.

### 9.2 Predicati ammessi

Sono ammessi i predicati sul valore del repertorio condiviso di §8.1 **quando la firma dell’occurrence e lo slot li autorizzano**, oltre ai nuovi confronti espliciti con un valore corrente del profilo. AND/OR possono essere annidati entro i limiti dei corpi di §7. Tutti i predicati dello stesso scope si riferiscono alla **stessa occurrence**. Un confronto su attributo sconosciuto o su JSON di kind non applicabile deve conservare l’incertezza per i quantificatori, non essere scartato preventivamente come fatto certamente falso. Le proiezioni temporali ammesse sono appartenenza a giorno/settimana/mese del calendario dichiarato e fascia oraria locale: non un repertorio di funzioni generiche sui timestamp.

Quando una occurrence possiede una lista strutturata, è ammessa una sola verifica esistenziale su un membro posseduto con predicati dello stesso membro. “Una riga con categoria Fotocamera e quantità ≥2” richiede un'unica riga con entrambi; due righe diverse non bastano. Non sono ammesse iterazioni annidate su liste arbitrarie.

La sorgente può essere un evento raw, un ruolo business che unifica eventi, oppure una evidenza normalizzata ammessa da un contratto. Le proiezioni alla grana riga, prodotto-per-ordine o altra evidenza dichiarata non sono automaticamente azioni aggiuntive. La grana del conteggio MUST essere dichiarata.

Una unione dichiarata di eventi conta ogni azione una volta. Se due selector comprendono la stessa identità business, l'azione non raddoppia. La correttezza richiede identità affidabile o un contratto equivalente; una somiglianza fra record non autorizza deduplicazione euristica.

### 9.3 Same-occurrence

“Fruizione dove categoria = Scienza AND dispositivo = mobile” esprime I022. Due scope indipendenti, uno Scienza e uno mobile, combinati con AND esprimono I023. Il builder MUST distinguere “aggiungi un requisito a questa attività” da “aggiungi un'altra condizione”.

### 9.4 Periodi e storia

Ogni scope MUST avere un periodo, anche quando la UI lo precompila. Sono distinti:

| Modalità | Significato |
|---|---|
| Periodo delimitato | Intervallo assoluto, mobile, di calendario o ancorato, con estremi dichiarati |
| Intera storia rilevante | Dall'inizio documentato della relazione pertinente fino a T; necessaria per “mai” e “primo di sempre” |
| Storia disponibile | Intervallo di storia conservata dichiarato dalla fonte, mostrato all'utente; non implica copertura antecedente e non è sinonimo di storia completa |

La storia disponibile non si espande implicitamente all'intera relazione. Un suo conteggio nullo può provare assenza **in quell'intervallo**, se l'intervallo è adeguatamente osservato; non prova “mai”. Eventuali buchi dentro l'intervallo continuano a richiedere UNKNOWN. Il primo evento osservato MUST NOT essere assunto come inizio della relazione o limite certificato di copertura.

Per una sorgente unificata, le lacune dei comportamenti componenti restano pertinenti: non si restringe silenziosamente la domanda all'intersezione dei periodi meglio osservati.

### 9.5 Ancore e vincoli di contesto

Uno scope può usare una data personale ammessa, come iscrizione o inizio del ciclo corrente. Può usare una occurrence soltanto se appartiene al contesto autorizzato che l'ha selezionata. Una chiave locale, quando richiesta, restringe tutte le occurrence dello scope a quella identità.

I predicati non possono leggere risultati di altre misure, eventi di altri scope, altre persone, relazioni libere o ancore create da condizioni sorelle. Lo scope non ammette aggregazioni interne al proprio `where`, subquery, selezione della “migliore riga” o raggruppamenti liberamente concatenati.

---

## 10. Typed comparisons and measures

### 10.1 Catalogo di riduzioni e test inline

Le capacità nella tabella sono **MUST inline** nei contesti ammessi. La tabella comprende sia riduzioni atomiche sia test terminali: non dichiara che tutti i risultati siano operandi. I confini operando/operazione sono definiti in §10.2. Alcune capacità hanno accesso mediante recipe, ma nessuna richiede un trait custom per essere costruita nella propria forma inline.

| Riduzione / test chiuso | Significato e valore business | Insieme certamente vuoto | Dati ignoti | Unità, denominatore e minimo | Accesso UX |
|---|---|---|---|---|---|
| Esistenza / assenza | Conteggio rispettivamente ≥1 o =0 delle azioni qualificanti | Esistenza: NO MATCH; assenza: MATCH, se la finestra è adeguatamente osservata | Una prova positiva può bastare; l'assenza richiede escludere occurrence qualificanti possibili | Azioni alla grana dichiarata; nessuna guardia di non-vuoto sull'assenza | “Ha fatto” / “non ha fatto” |
| Count | Numero di occurrence business distinte; almeno, al massimo, esattamente, intervallo | 0 | Limite inferiore certo e limite superiore possibile; non trasformare un conteggio parziale in un conteggio esatto | Intero non negativo; la grana rimane nel riepilogo | “Quante volte” |
| Sum | Somma dei valori della proprietà alle occurrence qualificanti | 0 nell'unità della misura | Nessun valore mancante è scartato o riempito con zero; eventuali prove da limiti sono ammesse secondo §19 | Unità uniforme e additiva; per denaro valuta confrontabile; minimo di occurrence opzionale | “Totale”, “ha speso”, “minuti”, “unità” |
| Average | Somma dei valori divisa per il numero di occurrence della stessa popolazione | Valore non definito; la guardia minima rende la condizione NO MATCH | Baseline: valore determinato soltanto con popolazione e valori adeguatamente noti; non media dei soli importi disponibili | Stessa unità della proprietà; minimo ≥1 obbligatorio, configurabile più alto | “Valore medio”; il minimo è nella stessa condizione |
| Min / max | Minimo/massimo valore numerico o temporale nello scope | Valore non definito; guardia minima non soddisfatta | Una soglia può essere provata da un valore testimone; gli altri casi richiedono evidenza sufficiente | Stessa unità; minimo ≥1 | “Valore minimo/massimo”, senza confonderlo con prima/ultima occurrence |
| Count distinct | Numero di valori distinti di una dimensione dichiarata | 0 | Valori ignoti non diventano una categoria “null”; possono rendere ignoto il risultato esatto | Dimensione esplicita: feature, prodotto, ordine, giorno ecc. | “Quanti diversi” |
| Share per occurrence | Numero di occurrence che soddisfano un predicato, diviso per tutte le occurrence dello scope base | Rapporto non definito; guardia di denominatore positivo fallisce | Le occurrence con proprietà ignota restano nel denominatore; la loro appartenenza al numeratore può essere incerta | Stessa base e periodo; quota in [0,1]; minimo di occurrence esplicito, almeno 1 | “Quale percentuale delle attività” |
| Share ponderata | Peso del sottoinsieme diviso per peso dell'intero scope | Non definita se peso totale =0 | Nessuna omissione di pesi mancanti; possibili limiti solo con peso non negativo e denominatore adeguato | Stesso peso additivo, non negativo; peso totale >0; minimo in minuti/importo/quantità ove richiesto | “Quota del tempo / del valore” |
| Rapporto fra misure | Rapporto fra due misure compatibili, non necessariamente sottoinsiemi | Non definito se denominatore =0 | Il rapporto conserva l'incertezza degli operandi; non produce infinito o zero artificiale | Denominatore noto positivo o intervallo certamente positivo; minimo contestuale configurabile | “Ticket ogni 10 utilizzi”, “quota consumata” |
| Confronto fra misure | Confronto diretto, differenza o variazione relativa nelle forme chiuse di §10.2 | Dipende dai valori: due conteggi vuoti valgono entrambi 0 | Nessun confronto numerico positivo è dedotto da operandi ignoti senza prova sufficiente | Unità compatibili; baseline >0 per variazioni percentuali; eventuale campione minimo | “Più acquisti che rimborsi”, “calo del 50%” |

Il prodotto MUST distinguere **valore non definito** da zero. Una guardia esplicita, per esempio “almeno due acquisti”, può essere certamente falsa e quindi rendere l'intera condizione NO MATCH anche quando la media non esiste.

### 10.2 Atomic scalar operands e terminal operations

#### 10.2.1 Operandi atomici ammessi

Un operando inline MUST essere uno scalare tipizzato ammesso nello slot che lo legge: **current profile scalar**, **scalar locale autorizzato**, **count**, **sum**, **average**, **min**, **max**, **count distinct**. Le costanti letterali compatibili sono parametri ammessi dei confronti. Un current profile scalar o un'altra evidenza scalare ammessa può essere un dato esterno derivato, purché ne siano dichiarate semantica e validità (§16); ciò non rende costruibile inline il calcolo che lo ha prodotto.

“Scalar locale autorizzato” significa soltanto una proprietà o un valore temporale del ruolo locale esplicitamente ammesso: per esempio, importo dell'ordine candidato, attributo della occurrence selezionata o durata nativa del preset temporale chiuso. MUST NOT significare qualunque risultato calcolabile sul contesto. La sola appartenenza allo stesso contesto non autorizza un operando.

Count, sum, average, min, max e count distinct riducono uno scope dichiarato a uno scalare; MUST NOT ricevere come argomento un'altra misura o formula. I predicati che qualificano lo scope MUST NOT leggere aggregati. Una riduzione futura MUST essere esplicitamente nominata, tipizzata e dotata di regole di grana, vuoto e incertezza prima di diventare ammessa.

#### 10.2.2 Forme terminali complete

| Forma chiusa | Parametri e semantica | Uscita |
|---|---|---|
| Confronto diretto | Due scalari atomici compatibili, oppure uno scalare e una costante, con operatore tipizzato | MATCH / NO MATCH / UNKNOWN |
| Difference | Due operandi atomici quantitativi compatibili A e B; differenza A meno B confrontata con una soglia costante nella stessa unità | Esclusivamente il risultato del test |
| Ratio | Due operandi atomici compatibili A e B; rapporto A/B confrontato con una soglia costante; B positivo | Esclusivamente il risultato del test |
| Relative change | Due operandi atomici, recente A e riferimento B, della stessa unità; variazione relativa (A−B)/B confrontata con una soglia percentuale costante; B positivo | Esclusivamente il risultato del test |
| Share | Una delle forme part/whole chiuse in §§10.3–10.4 e §12; quota confrontata con una soglia costante e con le guardie richieste | Esclusivamente il risultato del test |

La notazione aritmetica nella tabella definisce il significato matematico delle forme; MUST NOT essere esposta come una grammatica di formule. Il confronto con la soglia è parte della medesima forma terminale, non una seconda operazione concatenata. Una soglia aritmetica è un parametro letterale tipizzato, non un terzo operando calcolato. Le guardie di campione e validità sono requisiti booleani associati, non operandi aritmetici aggiuntivi.

**Difference, ratio e relative change MUST essere terminali. Il loro risultato MUST NOT diventare operando di un'altra operazione inline.** Lo stesso confine vale per la quota prodotta internamente da share. Al massimo due operandi atomici entrano in una forma aritmetica; avere soltanto due operandi non rende lecita un'operazione non elencata.

Il confronto moltiplicativo “dopo >2×prima” è una presentazione della ratio con soglia 2 e riferimento positivo, non un operatore generale di moltiplicazione. Un calo almeno del 50% significa relative change ≤−50%, con riferimento positivo e l'eventuale minimo richiesto. La differenza non impone un denominatore positivo perché non ha denominatore.

Il preset sugli ultimi due intervalli (§11.4) fornisce due durate tipizzate con estremi fissi: non legge risultati di operazioni difference configurabili. Le durate non diventano valori esportabili o nuove ancore; il preset non è un'eccezione che autorizza `difference(A,B)/C`.

#### 10.2.3 Esempi informativi di validità

| Forma | Stato e ragione |
|---|---|
| `COUNT(Purchase) > COUNT(Refund)` | Ammessa: confronto diretto di due conteggi, con grane dichiarate |
| `SUM(Purchase.amount) > profile.creditLimit` | Ammessa con importi e valuta confrontabili |
| `AVG(duration nel periodo A) > AVG(duration nel periodo B)` | Ammessa, con minimi e unità compatibili |
| `RefundCount / OrderCount < 10%` | Ammessa come ratio fra le due misure; non è automaticamente quota di ordini rimborsati |
| `SUM(Purchase.amount) − SUM(Refund.amount) > €1.000` | Ammessa come difference terminale |
| `(PurchaseCount + RefundCount) / SessionCount` | Vietata: addizione e composizione non previste |
| `MIN(A) + MAX(B) > X` | Vietata: l'addizione non è una forma ammessa |
| `ratio(A,B) > ratio(C,D)` | Vietata: usa due risultati terminali come operandi |
| `difference(A,B) / C` | Vietata: riusa un risultato terminale |

I nomi didattici degli operandi negli esempi non sono variabili nominabili dall'utente. “Measure” MUST significare una riduzione atomica autorizzata, non `Expression<Measure>`. Una recipe MUST NOT trasformare una formula vietata in uno pseudo-operando opaco.

Confrontare conteggi di acquisti e di operazioni di rimborso è ammesso anche se i repertori differiscono. Denaro e minuti non sono confrontabili; EUR e USD richiedono valori confrontabili già dichiarati. Rapporti che introducono unità composte arbitrarie non appartengono alla revisione 1.1 proposta.

### 10.3 Denominatori, scope e unità

Una **share** MUST derivare il numeratore dal medesimo scope base del denominatore, aggiungendo il solo predicato qualificante. Cambiare periodo, comportamento o grana crea invece un rapporto fra misure; non può essere ancora presentato come quota delle stesse occurrence.

Una **ratio** può confrontare scope distinti, ma il riepilogo MUST mostrare entrambi. Un confronto fra periodi MUST distinguere periodo di riferimento e periodo recente; la baseline non corregge automaticamente per la diversa durata. Un conteggio è un conteggio, non un tasso per giorno implicito.

Una verifica su una occurrence selezionata può richiedere insieme valuta EUR e importo >€100, dopo la selezione. La valuta fa parte dell'interpretazione del valore monetario; la validazione MUST NOT spostare quel requisito nella base di selezione per rendere più facile il confronto. Una currency nota diversa da EUR confuta il requisito EUR; una currency ignota non autorizza un confronto monetario privo di unità.

Gli importi del benchmark riportato dalla Fase 3 sono totali effettivamente pagati dopo sconti, inclusi tasse e spedizione, prima dei rimborsi. “In EUR” richiede valori confrontabili in EUR: MUST NOT significare scartare gli ordini in altre valute quando la domanda riguarda la spesa complessiva. Una conversione dichiarata a monte è ammissibile; una conversione implicita del builder no.

Per I039, i due scope sono acquisti del periodo e rimborsi del periodo, anche relativi a ordini precedenti. La riconciliazione dei soli ordini comprati nel periodo è un'altra domanda.

Per I095, il denominatore è la quota **attuale** del profilo, positiva e nella stessa unità del consumo. Il numeratore è il consumo del ciclo individuale corrente fino a T. Non si media la quota storica e non si limita artificialmente il rapporto al 100%.

Un minimo campione è un requisito della domanda, non una garanzia statistica. Il builder MUST conservarlo quando la recipe lo contiene e MUST NOT aggiungere un minimo maggiore di quello richiesto senza renderlo esplicito.

### 10.4 Distribuzioni, quote e prevalenza: forme chiuse

Le distribuzioni usano una sola dimensione business per operazione. Non esiste un costruttore libero di GROUP BY o una pipeline di raggruppamenti.

| Richiesta | Forma supportata | Regole |
|---|---|---|
| N valori distinti | V: count distinct | Una dimensione, grana e periodo dichiarati |
| Stesso valore ripetuto N volte | K: un valore con una misura locale ≥N | Si distingue conteggio delle occurrence da numero di ordini o giorni distinti |
| Almeno M valori, ciascuno ripetuto N volte | K: numero di identità che superano la soglia locale ≥M | Un solo livello per valore; nessun raggruppamento successivo arbitrario |
| Valore prevalente | V: unico valore più frequente | Almeno una occurrence; il massimo deve essere unico; una parità nota produce NO MATCH alla richiesta di vincitore unico |
| Quota di un valore/predicato | V: share per occurrence o per peso | Denominatore = tutto lo scope; ripetizioni incluse; peso esplicito |
| Nessun valore supera X% | V: quota massima per valore ≤X | Si considerano tutti i valori effettivi, non un elenco predefinito di categorie; minimo campione esplicito |
| Nuovo valore mai visto prima | K: valore recente e nessuna occurrence precedente con quella chiave | Confronto con la storia anteriore all'inizio del periodo recente, non con il solo periodo immediatamente precedente |
| Cambiamento della prevalenza | V: vincitore unico nei due periodi, confrontato o verificato contro due valori | Nessun vincitore arbitrario in caso di parità; periodi espliciti |

Per I041 la chiave esterna è prodotto; la misura locale conta **ordini distinti**. Tre unità o tre righe dello stesso prodotto in un solo ordine contano un ordine. Un ordine con più prodotti partecipa una volta per ciascun prodotto effettivo, senza moltiplicare il numero degli ordini del profilo.

Per I043 si contano i contenuti che hanno almeno due fruizioni. Per I059 la misura interna conta giorni distinti per canale: è una combinazione espressamente ammessa di chiave e misura locale, non una seconda dimensione liberamente navigabile.

**Prevalenza non significa maggioranza.** Con Scienza 45%, Arte 35% e Tecnologia 20%, Scienza è l'unico valore più frequente, ma non supera il 60%. La baseline definisce “prevalente” per frequenza. Quote ponderate sono già supportate; una diversa modalità di prevalenza ponderata MAY essere aggiunta con un nome esplicito, senza cambiare quella per frequenza.

**Parità e ignoti.** Ogni verifica sul vincitore, compreso il confronto fra vincitori di due periodi, contiene la guardia di unicità per ciascuna distribuzione. Una parità nota fa fallire tale guardia: non diventa un valore “diverso da Scienza”. Se i conteggi sono noti e Scienza è a pari merito, “Scienza è l'unica più frequente” è NO MATCH. Se categorie mancanti o occurrence non osservate possono cambiare il vincitore, è UNKNOWN. MUST NOT essere usato un tie-break alfabetico. Un valore ignoto non diventa una categoria reale denominata “sconosciuto”, salvo categoria business esplicitamente registrata e distinta dal null.

**Novità.** Per I061, sia W il periodo recente: esiste una categoria osservata in W senza osservazioni della stessa categoria prima dell'inizio di W. Una volta comparsa in W, ulteriori fruizioni in W non annullano la novità. I053 richiede invece che ogni contenuto fruito in W fosse già stato fruito prima di W, con almeno una fruizione recente. La storia precedente e la qualificazione delle occurrence devono permettere di decidere queste assenze o presenze.

Le operazioni “nuovo” e “già noto” non restituiscono un insieme riutilizzabile da un'altra condizione: la chiave resta locale.

Le verifiche su vincitore unico, cambiamento di prevalenza e quota massima per valore sono **test terminali nominati**, non ulteriori operandi atomici. Il vincitore resta interno agli slot “confronta con un valore” o “confronta i vincitori dei due periodi” già definiti; MUST NOT essere usato come chiave di un altro scope. La quota massima per valore MUST mantenere base, dimensione, grana e denominatore comuni; MUST NOT alimentare altra aritmetica.

Le forme di ripetizione e novità riportate nella tabella sono applicazioni dell'Insieme per identità (§12), con i soli slot di quella forma. Non autorizzano un corpo più ampio per tutte le distribuzioni.

### 10.5 Guardie e valori non definiti

Le guardie sono valutate con la logica di §20 insieme al test principale. Denominatore certamente zero o negativo fa fallire la guardia di positività della ratio; denominatore ignoto non viene sostituito con uno e può produrre UNKNOWN. Una share richiede un denominatore positivo e un minimo almeno pari a uno, oltre alle eventuali soglie di peso. Average e min/max richiedono la popolazione minima dichiarata; il fallimento certo della guardia produce NO MATCH anche se il valore numerico non è definito. L'incertezza quantitativa segue esclusivamente §19.

Per la **share di oggetti/opportunità**, il numeratore conta candidati della stessa base che soddisfano il corpo e il denominatore conta tutti i candidati della base, non le loro occurrence di esito. Due rimborsi sul medesimo ordine possono contare due operazioni in una ratio, ma un solo ordine rimborsato nella share di ordini. Periodi, quantificatori e appartenenza part/whole MUST essere preservati; share MUST NOT diventare un alias di arbitrary ratio.

---

## 11. Selected experiences

### 11.1 Procedura normativa

La selezione ha tre stadi, sempre distinguibili:

> **Base di selezione → posizione nell'ordine temporale → verifiche sull'esperienza scelta.**

La base contiene repertorio, prefiltri della stessa occurrence e periodo di selezione. Le verifiche successive possono riguardare proprietà, recency, ritardo o attività nelle finestre autorizzate intorno alla occurrence scelta. Un fallimento successivo MUST NOT provocare la ricerca di una occurrence alternativa che passi i requisiti.

### 11.2 Selettori ammessi

| Selettore | Significato | Requisito di osservazione |
|---|---|---|
| Primo di sempre | Prima occurrence nella relazione rilevante | Nessuna occurrence precedente possibile rispetto all'inizio documentato |
| Primo nel periodo | Prima occurrence nella base temporale delimitata | Adeguata osservazione dall'inizio di quel periodo alla occurrence scelta |
| Primo qualificante | Primo dopo aver applicato i prefiltri dichiarati, nella base scelta | Anche i possibili qualificanti precedenti devono essere esclusi |
| Ultimo assoluto | Ultima occurrence del repertorio fino a T, senza prefiltri di proprietà non richiesti | Adeguata osservazione dalla occurrence candidata a T; la mancanza di storia più antica non la rende automaticamente ignota |
| Ultimo qualificante | Ultimo nella base dopo i prefiltri dichiarati | Nessun qualificante successivo possibile |
| N-esimo | N-esima occurrence dall'inizio della base; **N da 1 a 10** | Prefisso e posizione adeguatamente osservati |
| Ultimi N | Blocco delle N occurrence più recenti; **N da 1 a 10** | Devono esistere N occurrence e il blocco deve essere identificabile |

Il periodo può essere l'intera storia, un intervallo esplicito o la storia disponibile dichiarata. Il nome “primo di sempre” è riservato all'intera relazione. La selezione cronologica non ammette ordinamento per importo, voto, durata, nome o identificatore tecnico.

Se la base certamente non contiene la occurrence richiesta, la condizione sull'esperienza è NO MATCH. Se non sappiamo se esista o quale sia, è UNKNOWN. In particolare, una storia completa con due Export non soddisfa “il terzo Export”; una storia incompleta con due Export non permette la stessa conclusione.

Una parità temporale non risolta da un ordine affidabile MUST NOT essere sciolta tramite ordine di ricezione o ID casuale. Se le selezioni compatibili con l'ordine noto possono produrre esiti diversi, la condizione è UNKNOWN. Se tutte producono lo stesso esito, questo è determinabile senza inventare quale occurrence sia prima: la spiegazione deve conservare l'ambiguità del testimone. L'ultimo valore non-null non sostituisce mai il valore dell'ultima occurrence.

### 11.3 Contrasti normativi

| Richiesta | Prima della selezione | Selezione | Dopo la selezione |
|---|---|---|---|
| I025: primo acquisto di sempre, recente e EUR | Tutti i Purchase nella storia completa | Primo | Data negli ultimi 30 giorni; valuta EUR |
| I026: primo acquisto in EUR, recente | Purchase con valuta EUR nella storia completa | Primo | Data negli ultimi 30 giorni |
| I027: primo acquisto del periodo, EUR | Purchase negli ultimi 30 giorni | Primo | Valuta EUR |
| I028: ultimo acquisto recente, EUR, >€100 | Tutti i Purchase fino a T | Ultimo | Data negli ultimi 90 giorni; valuta EUR; importo >€100 |
| I029: ultimo acquisto EUR recente e >€100 | Purchase con valuta EUR fino a T | Ultimo | Data negli ultimi 90 giorni; importo >€100 |
| “Ultimo acquisto con amount >100” | Purchase con amount >100 | Ultimo | Soltanto gli eventuali ulteriori requisiti dichiarati |

I030 non seleziona l'ultimo: richiede una qualsiasi occurrence EUR >€100 nel periodo.

### 11.4 Confronti finiti sulle ultime esperienze

La baseline MUST offrire tre configurazioni chiuse aggiuntive:

**Ultimo rispetto al precedente.** Confronta lo stesso attributo nelle ultime due occurrence dello stesso repertorio: uguaglianza/diversità, differenza numerica o rapporto con denominatore positivo. I031 usa calo di almeno 3 punti NPS. Entrambe le occurrence devono esistere; non è consentito scegliere due posizioni arbitrarie o due campi arbitrari di eventi diversi.

**Predicato sulle ultime N.** Verifica alcune, tutte o un numero di occurrence del blocco selezionato usando lo stesso predicato. “Gli ultimi tre NPS sono tutti ≤6” seleziona prima tre NPS, non tre NPS già bassi.

**Confronto degli ultimi due intervalli.** Preset fisso sugli ultimi tre eventi: due intervalli temporali positivi, rapporto fra l'intervallo più recente e il precedente. Copre I093 senza aprire statistiche sulle serie di intervalli. Mediana, intervallo “tipico” e confronto N-esimo–M-esimo restano fuori dalla baseline.

### 11.5 Ancore locali

I035 seleziona il primo Export storico e ne verifica il momento rispetto all'iscrizione; non seleziona il primo Export già dentro la finestra di sette giorni. I036 sceglie l'ultimo upgrade e verifica Export nei sette giorni precedenti. I037 sceglie la creazione iniziale, poi il primo contatto effettivamente successivo, e richiede ritardo >48 ore: nessun contatto non equivale a un contatto tardivo.

Per I096 si sceglie prima l'ultimo upgrade, poi si verificano recency, minimo precedente e confronto fra le due finestre ancorate a quell'upgrade. La finestra recente non può spostarsi su un altro upgrade che produce un confronto più favorevole.

---

## 12. Identity sets and quantified candidates

### 12.1 Candidati identificabili e corpi autorizzati

La baseline MUST supportare K indipendentemente da P. La coesione normativa è: **una fonte produce candidati identificabili; un corpo autorizzato valuta ciascun candidato; un quantificatore determina il risultato**. Non è un'equivalenza algebrica fra grouping, correlation e authoritative state. Le forme e gli slot sono distinti secondo §7.2 e §15.

La forma per la correlazione senza ordine è:

> Fra le identità introdotte da uno scope seme o da una fonte autorevole ammessa, verifica un corpo locale per ciascuna identità e applica un quantificatore al risultato.

Nella modalità di identità correlata, il corpo può usare fino a quattro scope foglia, tutti ristretti alla stessa chiave, e un pivot primo/ultimo della sorgente primaria quando previsto. Le finestre sono indipendenti ed esplicite, oppure ancorate al pivot autorizzato. Non si possono unire flussi mediante condizioni diverse dall'identità semantica dichiarata. Le situazioni autorevoli usano invece soltanto gli slot specifici del proprio contratto: per esempio, il trial può ancorare l'assenza di upgrade senza richiedere un trial ID su ogni upgrade.

Esempio: “esiste un prodotto visualizzato negli ultimi 30 giorni ma mai acquistato” introduce i product ID delle visualizzazioni recenti e verifica zero acquisti di ciascuno nell'intera storia rilevante. Un acquisto di quel prodotto **precedente** alla visualizzazione esclude quel prodotto. La forma “visualizzato e non acquistato dopo la visualizzazione” può invece includerlo: non sono la stessa domanda.

### 12.2 Identità, grana e candidati

**Dimensione osservata e identità di oggetto non sono intercambiabili.** Per ripetizione e novità di un valore, K usa una dimensione del medesimo repertorio, eventualmente su periodi differenti: categoria di fruizione, canale di interazione, valore di una proprietà. Questa forma non autorizza a collegare eventi arbitrari perché due loro numeri o testi coincidono. Per correlare repertori differenti serve invece un ruolo di identità business comune dichiarato, come product, order o message. Una semplice uguaglianza di prezzo non diventa una identità di oggetto.

La chiave MUST avere dominio e namespace dichiarati; il valore testuale uguale in namespace differenti non basta. Una chiave semantica può avere una composizione interna definita dal catalogo, ma l'utente non costruisce confronti fra campi o congiunzioni di join.

Se il mapping è assente a livello strutturale, la condizione correlata non è attivabile. Se manca la chiave soltanto in alcune occurrence, queste rimangono possibili evidenze irrisolte: possono impedire di provare un'assenza o un “tutti”. Non si scartano silenziosamente.

La fonte dei candidati si definisce **prima** del corpo verificato. “Tutti gli ordini recenti sono senza rimborsi” non può prima selezionare soltanto gli ordini senza rimborsi. Un filtro su una proprietà del seme è invece ammesso quando appartiene alla domanda: “gli ordini con fotocamera comprati nel periodo”.

### 12.3 Quantificatori

| Quantificatore | Significato | Insieme candidato certamente vuoto |
|---|---|---|
| Alcuni / almeno uno | Esiste un candidato che soddisfa il corpo | NO MATCH |
| Nessuno | Nessun candidato soddisfa il corpo | MATCH |
| Cardinalità: almeno / al massimo / esattamente N / intervallo | Numero di candidati che soddisfano il corpo; estremi cardinali inclusi per “tra L e U” | Si confronta 0 con la soglia o con l’intervallo |
| Tutti | Tutti i candidati soddisfano il corpo **e ne esiste almeno uno** | NO MATCH |
| Tutti, minimo N | Come sopra, con almeno N candidati | NO MATCH |
| Quota di candidati | Candidati con corpo soddisfatto / candidati della base | Non definita; guardia di non-vuoto fallisce |

I candidati con corpo UNKNOWN non spariscono dal denominatore. Un controesempio certo può confutare “tutti”; un testimone certo può provare “alcuni”, anche se altri candidati non sono determinabili. Per dimostrare “nessuno” serve invece evidenza sufficiente su tutti i possibili candidati.

“Non esiste un'opportunità con esito X” e “tutte le opportunità hanno l'esito opposto” differiscono almeno per il non-vuoto; non sono etichette intercambiabili.

### 12.4 Opportunità distinte e loro selezione

Un'opportunità ha una chiave distinta e un inizio riconoscibile. Le forme della baseline sono prima consegna del messaggio, prima consegna della campagna al profilo, apertura iniziale del ticket nel ruolo dichiarato, oppure un evento iniziale univoco per una chiave di dominio.

La costruzione MUST seguire quest'ordine:

> Identifica le opportunità e il loro inizio → applica il periodo/qualificatori delle opportunità → eventualmente seleziona le ultime N → valuta gli esiti.

La prima consegna si identifica rispetto alla storia pertinente della stessa chiave, non soltanto fra le consegne cadute nel periodo. Una riconsegna recente di un messaggio già consegnato prima non crea automaticamente una nuova opportunità.

Le **ultime N opportunità**, con N da 1 a 10, sono ordinate per il loro inizio. Servono almeno N opportunità. Ogni inizio deve essere correttamente identificato: una coda recente completa non basta, da sola, a stabilire le prime consegne storiche delle campagne. Un vecchio inizio non osservato può cambiare l'ordinamento delle opportunità anche quando tutte le riconsegne recenti sono note. In I099 si scelgono le ultime tre campagne distinte per prima consegna, non le ultime tre email né le ultime tre campagne cliccate. Ogni campagna vale una volta; il click deve essere successivo alla sua consegna e precedente a T.

In I048 ogni messaggio vale una volta anche con molti click. Le opportunità dell'esempio sono tutte maturate per costruzione del periodo. In generale un'opportunità immatura conserva esito UNKNOWN; MUST NOT essere rimossa dal denominatore senza una scelta esplicita della base “sole opportunità mature”. Una quota può essere già dimostrabile dai suoi limiti, senza trasformare i singoli esiti pendenti in esiti definitivi.

### 12.5 Ordini e rimborsi

I076 richiede almeno un ordine comprato nel periodo senza alcuna operazione di rimborso relativa fino a T. I077 richiede almeno un ordine nel periodo e assenza di rimborsi per **tutti** quegli ordini. Un ordine rimborsato e uno non rimborsato distinguono i due casi.

“Fino a oggi, per quell'ordine” non introduce da solo una precedenza aggiuntiva: l'identità e l'orizzonte prescritti sono sufficienti. Una richiesta esplicitamente “dopo l'acquisto” aggiunge invece l'ordine temporale.

I050 usa un ordine del periodo, il suo totale pagato e la somma dei rimborsi dello stesso ordine fino a T. La somma deve raggiungere almeno il totale pagato; non basta un evento Refund. Il confronto non è una ratio e quindi non introduce una guardia di denominatore positivo. La specifica non aggiunge condizioni non presenti nel benchmark, come un importo minimo del pagato o un numero minimo implicito di operazioni di rimborso.

### 12.6 Evidenza dei candidati e limiti di cardinalità

Un candidato conta una sola volta alla propria identità, anche quando più occurrence ne dimostrano l'appartenenza o più esiti ne soddisfano il corpo. La base viene determinata prima della verifica; i candidati con corpo UNKNOWN MUST NOT essere eliminati.

Con inventario candidato completo, S candidati certamente qualificanti e U incerti, il numero qualificante è compreso fra S e S+U. Un minimo N è MATCH se S≥N, NO MATCH se S+U<N, UNKNOWN altrimenti. Per un intervallo inclusivo [L,H], è MATCH se tutti i conteggi possibili appartengono all'intervallo, NO MATCH se nessuno vi appartiene, UNKNOWN negli altri casi. Un inventario incompleto può aggiungere candidati possibili e richiede i limiti certificati di §19; MUST NOT essere trattato come inventario completo.

Per “tutti” la guardia di almeno un candidato, o di almeno N quando esplicita, resta obbligatoria. L'assenza di un candidato qualificante non costituisce il complement di un'espressione arbitraria: si applica soltanto al corpo autorizzato della forma K scelta.

---

## 13. Ordered paths

### 13.1 Sei richieste non equivalenti

| Forma | Semantica |
|---|---|
| A AND B | Esiste A ed esiste B; occurrence indipendenti; nessun ordine o oggetto comune |
| A THEN B | Esiste una coppia distinta in cui A precede B |
| A THEN B within X | Esiste una coppia ordinata con ritardo non superiore a X, limite massimo incluso |
| A THEN B on same object | Come sopra, con una sola identità comune effettiva fra le occurrence |
| A WITHOUT B within X | Esiste un'occasione A per cui B non si verifica nella finestra successiva prescritta; l'assenza positiva richiede finestra maturata e osservata |
| A WITHOUT B on same object | Deve essere completata con un orizzonte: nessun B su quell'oggetto nell'intervallo dichiarato, oppure nessun B dopo A; le due forme sono distinte |

L'ultima frase, da sola, non determina un filtro completo. Il builder MUST chiedere di scegliere il significato attraverso opzioni esplicite, non dedurlo dal nome “abbandono”.

### 13.2 Precedenza, ordine libero negli intervalli e adiacenza

Indichiamo con **a ≺ b** una precedenza effettiva fra due occurrence distinte. Un timestamp precedente la prova; a timestamp uguali serve un ordine ulteriore affidabile del dominio. Un ordinamento tecnico arbitrario non la prova.

**Percorso normale:** richiede a ≺ b ma ammette altre azioni intermedie. È il significato di “poi”.

**Prima azione successiva:** oltre alla precedenza, richiede che non esista alcuna azione del repertorio dichiarato strettamente fra a e b. Questa è la modalità di adiacenza, non un ordinamento “più vero” degli stessi timestamp. Il repertorio delle azioni considerate MUST essere esplicito e adeguatamente osservato; le azioni automatiche non entrano quando il repertorio le esclude. Copre I071.

Un ritardo è misurato fra i tempi delle azioni. A parità di timestamp, una precedenza affidabile può avere ritardo zero. Non si riusa la medesima occurrence per soddisfare due passi dello stesso percorso.

### 13.3 Forma del percorso positivo

Un percorso MUST avere da 2 a 5 passi. Ogni passo ha uno scope e può usare “una occurrence qualsiasi” oppure un selettore primo/ultimo nella sua base esplicita. Il selettore si applica **prima** della ricerca del percorso; non significa “primo che rende valido il percorso”. In P, questa selezione è sulla base del profilo dichiarata, non implicitamente per oggetto. La selezione per chiave appartiene a K.

Sono configurabili un limite di durata totale dal primo all'ultimo passo e limiti fra passi adiacenti. Eventuali minimi di ritardo sono confronti di durata espliciti. Un percorso con limiti incompatibili è una configurazione invalida, non un segmento vuoto valutato normalmente.

Il periodo è esplicito: **tutti i passi nel periodo** è la configurazione normale; sono ammesse finestre dichiarate sui singoli passi quando la domanda lo richiede. I065–I069 impongono il periodo a tutti i passi. I071 lo impone al contatto con l'assistenza; l'errore può precedere il bordo iniziale entro i dieci minuti ammessi. MUST NOT essere aggiunta una restrizione globale che cambia questo caso.

L'esistenza di **qualsiasi tupla interamente valida** soddisfa il percorso. Il prodotto MUST NOT usare una scelta greedy del primo A o del primo B che può perdere un'altra tupla valida. La selezione esplicita dell'ultimo click, invece, impedisce il recupero di un click precedente: è proprio la differenza fra I067 e I068.

### 13.4 Scoped correlation block

Un percorso MUST scegliere una sola delle forme seguenti: **nessuna correlation key**; **una singola semantic correlation key su tutti i passi**; oppure **una singola semantic correlation key su un solo blocco contiguo di almeno due passi**. La forma globale è il caso in cui il blocco coincide con l'intero percorso, non un secondo blocco.

I passi del blocco MUST condividere lo stesso valore semantico della chiave, con dominio e namespace dichiarati. I passi esterni MUST NOT ereditarlo: non sono ristretti da quella chiave, non possono leggerla nei predicati e non ne cambiano il valore. Nessuna chiave è esposta come variabile nominabile.

MUST esserci al massimo una correlation key e al massimo un correlation block. Il blocco MUST essere contiguo. MUST NOT essere possibile aggiungere una seconda chiave, correlare archi diversi tramite chiavi diverse, definire formule di join o richiedere lo stesso valore su due blocchi separati.

Per chiavi multivalore dichiarate deve esistere un singolo valore comune a **tutti i passi del blocco**. Ogni prova mantiene lo stesso valore anche nelle foglie autorizzate a leggerlo: MUST NOT scegliere valori diversi per rendere veri requisiti diversi della medesima prova. La ricerca è esistenziale sulla tupla e sul valore comune, senza esporre tale valore come variabile. Intersezioni soltanto pairwise non bastano. I passi esterni non partecipano al calcolo di quell'intersezione. Una chiave mancante nel blocco segue le regole di evidenza; una chiave mancante fuori dal blocco non rende da sola incerta la correlazione.

**Esempio informativo — O03.** Primo Purchase di sempre → Refund dello stesso order ID → nuovo Purchase. Il blocco correlato comprende i primi due passi; il terzo richiede soltanto una nuova occurrence successiva al rimborso. Il primo Purchase è selezionato sulla propria base storica prima della ricerca della tupla. Non viene sostituito da un altro ordine che abbia un rimborso più favorevole.

**Esempio informativo escluso.** A → B per product ID, B → C per cart ID, C → D per order ID non è una configurazione valida. Non è un percorso che restituisce NO MATCH: è una forma fuori dal linguaggio.

#### Assenze locali e accesso alla chiave

Sono ammessi predicati di assenza fra passi adiacenti o fra primo e ultimo. L'intervallo è strettamente fra le occurrence, usando l'ordine affidabile anche a timestamp uguali. Senza restrizione di identità, lo scope negativo riguarda il repertorio del profilo esplicitamente scelto.

Uno scope negativo può leggere la chiave locale soltanto quando entrambi gli estremi dell'intervallo appartengono al blocco correlato. Un esito dal passo finale può leggerla soltanto se il passo finale appartiene al blocco. Questi slot ristretti alla stessa identità MUST usare l'unica chiave del blocco: non creano un'altra correlazione e non possono estenderne l'ambito. Per correlare un percorso positivo e i suoi esiti, il blocco dei passi positivi deve già rispettare il minimo di due passi; un esito negativo non costituisce da solo un secondo passo del blocco.

I070 verifica zero errori dello stesso progetto fra creazione e condivisione scelte; errori precedenti o successivi non contano. MUST NOT essere disponibili formule fra proprietà arbitrarie dei passi. Identità nell'ambito autorizzato, ordine, durata, predicati sui singoli passi e gli esiti enumerati esauriscono le relazioni ammesse.

### 13.5 Esiti dopo un'ancora

Un esito positivo o negativo può essere valutato dopo una occurrence selezionata, un'opportunità, oppure il passo finale di un percorso. Gli orizzonti sono:

| Orizzonte | Intervallo dell'esito | Regola |
|---|---|---|
| Entro Δ dopo l'ancora | Occurrence effettivamente successive, con ritardo ≤Δ | Il limite massimo è incluso |
| Da allora fino a T | Occurrence effettivamente successive, con tempo < T | Non c'è una deadline chiusa diversa da T |
| Fra due passi scelti | Occurrence strettamente intermedie | È una proprietà del percorso scelto, non dell'intera storia |

Un esito negativo entro Δ è UNKNOWN finché la finestra non è maturata, salvo una prova positiva di risposta che lo rende già NO MATCH. Il periodo che seleziona le occasioni non tronca il periodo di ricerca della risposta.

Il momento di valutazione esclude gli eventi con timestamp T. Di conseguenza, se la deadline d coincide esattamente con T, la sua estremità inclusa non è ancora interamente osservata: la mancata risposta rimane UNKNOWN. La maturazione negativa richiede **T > d** e copertura certificata fino a d inclusa. Questo elimina un'ambiguità reale di bordo senza cambiare “entro”, che continua a includere il limite.

### 13.6 Eventi tardivi, più match e conteggio

Un click avvenuto all'ottavo giorno non cambia l'esito “nessun click entro sette giorni”; cambia invece “non ha ancora cliccato”. Un click avvenuto al sesto giorno ma ricevuto tardi MUST rettificare l'evidenza della deadline. Un risultato basato su dati correggibili non è un verdetto immutabile sul passato.

Più match possibili non sono consumati o attribuiti in esclusiva. Una occurrence può contribuire a più prove diverse quando soddisfa le relazioni dichiarate. Il prodotto non sta facendo attribuzione marketing.

La baseline **non conta tutte le combinazioni di sequenza** e non offre una politica implicita di match non sovrapposti. P produce esistenza. Per frequenze e percentuali di risposta si contano opportunità distinte in K, una volta ciascuna; per i ritorni si contano episodi canonici in T. I079 usa quattro passi positivi sullo stesso ticket, non un contatore ricorsivo di workflow.

### 13.7 Risultato, evidenza derivata e confine del percorso

P determina l'esistenza di una tupla completa valida. Una tupla certa basta per MATCH; se tutte le tuple possibili sono escluse il risultato è NO MATCH; se identità, ordine, selezione, assenze o proprietà possono cambiare l'esistenza di una tupla valida, il risultato è UNKNOWN. Un controesempio a una tupla non confuta altre tuple, salvo la selezione first/last esplicitamente vincolante.

I passi MUST poter consumare certified occurrences o milestone conformi a §16, quando disponibili, e verificarne proprietà certificate come normali proprietà del passo. La primitiva MUST NOT conoscere algoritmi di produzione, porzioni di contenuto o slot verticali. Un fatto storico associato alla milestone deve essere già certificato nell'evidenza: il percorso non lo ricostruisce mediante lookup storico.

I vincoli negativi o gli esiti sono foglie della costruzione, non arbitrary path expressions. Branching, quantificazioni su sotto-percorsi, conteggio delle combinazioni e processi ricorsivi rimangono esclusi.

---

## 14. Activity patterns

Lo **Schema di attività** MUST essere interpretato come un **catalogo chiuso di temporal activity recognizers**, non come un algoritmo temporale universale. Un pattern è una forma nominata con parametri, grana temporale, regole di bordo ed evidenza dichiarate. Nuovi pattern richiedono una nuova definizione semantica esplicita.

### 14.1 Catalogo chiuso della baseline

| Pattern | Stato di prodotto | Significato normativo |
|---|---|---|
| Attivo in N giorni distinti | MUST inline | Numero di giorni del calendario scelto con almeno un'attività nello scope |
| Attivo in ogni settimana | MUST inline | Ogni settimana richiesta contiene almeno un'attività; nessuna compensazione fra settimane |
| Attivo in almeno N degli ultimi M periodi | MUST inline | Cardinalità di periodi calendar con almeno un'attività, senza eliminare periodi vuoti |
| Qualified-period cardinality | MUST inline | Numero di periodi di una serie finita che soddisfano un test temporale chiuso ammesso in §14.7 |
| Attivo almeno N giorni per settimana | MUST inline, configurazione ulteriore | Ogni settimana richiesta contiene N giorni distinti attivi |
| N eventi in un giorno | MUST inline | Esiste un singolo giorno di calendario nel periodo con conteggio ≥N |
| N eventi in una finestra mobile continua | MUST inline, guidato | Esiste un intervallo continuo di durata fissa interamente contenuto nel periodo, con conteggio ≥N |
| Giorni consecutivi attivi | MUST inline | Esiste una serie di almeno N giorni di calendario consecutivi attivi, interamente nel periodo |
| Nessun gap maggiore di X | MUST inline, guidato | Il massimo intervallo senza attività, bordi del periodo inclusi, non supera X |
| Aumento/calo fra periodi | MUST, recipe su V | Confronto fra misure dello stesso repertorio; baseline e soglia minima esplicite |
| Ha smesso | MUST, macro | Presenza nel periodo precedente e assenza nel recente; eventuale attività residua come requisito distinto |
| È tornato dopo inattività | MUST inline tramite recipe T | Un ritorno reale preceduto da uso, dopo una pausa minima osservata |
| Ritorni ripetuti | MUST, riuso di T | Numero di pause/ritorni distinti, non numero di coppie possibili fra eventi |
| Rallentamento degli ultimi intervalli | MUST, preset E | Ultimi tre eventi e due intervalli positivi; non un linguaggio di time series |
| Cadenza personale tipica / mediana | MAY successivamente, principalmente derived | Non disponibile nella baseline; nessuna sostituzione mediante media o soglia fissa |

Questa scelta mantiene inline i pattern di resa elevata e alcune varianti ADVANCED che riusano gli stessi dati. Non introduce espressioni universali sulle serie temporali.

### 14.2 Calendario e giorni distinti

I giorni e le settimane usano il calendario esplicito del filtro. I080 conta quattro giorni nelle ultime quattro settimane complete: possono stare nella stessa settimana. I081 richiede attività in tutte e quattro le settimane. I082 richiede almeno tre giorni distinti in ciascuna; dodici giorni distribuiti diversamente non bastano.

Per “giorni completi” e “settimane complete” il periodo esclude il giorno o la settimana in corso. Un profilo iscritto dopo l'inizio di una delle settimane richieste non ottiene settimane attive inventate: una settimana realmente senza attività è un controesempio; una settimana non osservabile può produrre UNKNOWN.

### 14.3 Burst: calendario contro finestra continua

“N in un giorno” considera i giorni del calendario, di durata reale anche diversa da 24 ore quando il calendario lo comporta. “N in qualsiasi 24 ore” considera intervalli di **24 ore trascorse**, con inizio variabile, nella forma [inizio, inizio +24 ore), interamente contenuti nel periodo esterno.

Per I083 il periodo esterno sono gli ultimi sette giorni completi. Tre avvii alle 23 e due alle 01 possono soddisfare il burst continuo senza soddisfare I084. Eventi distanti esattamente 24 ore non appartengono insieme alla stessa finestra continua semiaperta di 24 ore. Questo operatore non usa la convenzione inclusiva di “entro una deadline dopo un evento”.

### 14.4 Continuità e gap ai bordi

Per uno scope in [s,e), con tempi ordinati t1…tn, il massimo gap è il massimo fra il vuoto iniziale s→t1, i vuoti fra tempi consecutivi e il vuoto finale tn→e. Se non esistono attività, il gap è e−s. L'assenza di dati intermedi può impedire di provare che un gap sia realmente privo di attività.

I086 valuta anche i due bordi. Il pattern “nessun gap oltre sette giorni” non equivale a contare giorni attivi o a controllare soltanto intervalli fra eventi noti. Una durata uguale a sette giorni soddisfa il limite “non oltre sette”.

### 14.5 Definizione canonica di ritorno

Siano u e r due utilizzi consecutivi nel repertorio considerato, con u ≺ r, ritardo almeno Δ e nessun utilizzo intermedio. **r è un ritorno dopo una pausa di almeno Δ**. Devono essere documentati sia l'uso precedente sia l'assenza durante la pausa. La prima attività in assoluto non è un ritorno.

Per provare un ritorno non serve necessariamente la storia completa di tutta la vita: servono un predecessore reale e copertura adeguata della pausa. Un semplice inizio della retention non costituisce il predecessore.

Ogni r identifica un solo episodio; più eventi antecedenti non moltiplicano i ritorni. Quando si contano ritorni con pause interamente in un periodo, sia u sia r e l'intera pausa devono appartenere al periodo richiesto. Questo copre I092 senza conteggio combinatorio.

I090 richiede un ritorno negli ultimi sette giorni dopo almeno trenta giorni senza uso. Per S011 si seleziona **l'ultimo ritorno qualificante**, si verifica che sia recente e, nello stesso contesto, che non sia stata ricevuta la campagna Bentornato dopo quel ritorno. Il totale dei minuti negli ultimi sette giorni rimane una misura indipendente: non viene ristretto implicitamente al tempo dopo il ritorno.

### 14.6 Cambiamenti composti

I087 confronta i due periodi di quattordici giorni, con almeno dieci utilizzi nel precedente. I088 richiede due diminuzioni consecutive fra tre settimane complete, non soltanto che l'ultima sia inferiore alla prima. I089 richiede uso precedente di Export, zero Export recente e almeno tre utilizzi di altre funzionalità recenti. I091 richiede assenza nell'ultima settimana completa e presenza in ognuna delle quattro precedenti.

Questi casi sono macro su misure e pattern, non nuovi operatori “churn”, “engagement” o “adoption” dal significato opaco.

### 14.7 Qualified-period cardinality

La costruzione MUST valutare una **serie finita di M periodi calendar dichiarati**, applicare separatamente a ciascun periodo il medesimo test temporale chiuso autorizzato, contare quanti periodi qualificano e confrontare il numero con una cardinalità. La serie viene definita prima del test. I periodi hanno lo stesso tipo e calendario, sono consecutivi e non sovrapposti; nessun periodo viene generato a partire dall'esito di un altro.

Nella revisione 1.1 proposta i tipi di periodo sono giorno, settimana e mese del catalogo temporale (§17). La selezione è “ultimi M periodi completi”, con M intero positivo, oppure una serie finita di periodi completi consecutivi delimitata esplicitamente. I periodi incompleti correnti non sono inclusi implicitamente. Non sono ammesse liste scelte eliminando i periodi sfavorevoli.

Il test per ogni periodo MUST appartenere al seguente sotto-catalogo chiuso. In tutti i casi il repertorio e i predicati same-occurrence sono fissi per la costruzione, e il pattern viene valutato entro i bordi del singolo periodo.

| Test qualificante ammesso | Semantica nel periodo |
|---|---|
| Periodo attivo | Almeno una occurrence qualificante |
| Soglia di giorni attivi distinti | Almeno D giorni del calendario con attività, con D intero positivo |
| Burst in giorno calendar | Esiste un giorno del periodo con almeno Q occurrence |
| Rolling burst | Esiste una finestra di durata fissa interamente contenuta nel periodo con almeno Q occurrence |
| Streak | Esistono almeno D giorni attivi consecutivi interamente contenuti nel periodo |
| Max gap | Il massimo intervallo senza attività, bordi inclusi, non supera la durata fissata |

Questi sono recognizer già definiti in §§14.1–14.4, non predicati liberi sulla serie. I pattern di ritorno, confronto fra periodi, stopped/resumed e increase/decrease conservano le proprie forme separate; non sono automaticamente corpi annidabili di qualified-period cardinality. Una qualified-period cardinality MUST NOT contenerne un'altra, combinare recognizer tramite formule, raggruppare di nuovo i bucket o diventare un generico GROUP BY temporale.

Sono MUST almeno le soglie **almeno N**, **al massimo N**, **esattamente N** e **tra L e H inclusi**, coerenti con la cardinalità generale. N, L e H sono interi non negativi; i vincoli devono essere staticamente compatibili con M e fra loro. Una richiesta “almeno N” con N>M è configurazione invalida; non viene reinterpretata diminuendo N. “Ogni periodo” richiede che tutti i M periodi qualifichino; M è sempre positivo.

Un periodo certamente privo di attività è comunque valutato dal test: per “almeno D giorni attivi” fallisce; per max gap il gap coincide con l'intera durata del periodo. **Le finestre vuote MUST NOT essere eliminate implicitamente.** Un periodo non osservabile può produrre UNKNOWN secondo §19; non viene rimosso né sostituito con uno antecedente.

Siano S i periodi certamente qualificanti, F i certamente non qualificanti e U gli incerti, con S+F+U=M. La cardinalità è compresa fra S e S+U. Per “almeno N”, MATCH se S≥N, NO MATCH se S+U<N, UNKNOWN altrimenti. Per l'intervallo [L,H], MATCH se [S,S+U] è interamente incluso, NO MATCH se è disgiunto, UNKNOWN negli altri casi. Questa conta dei risultati individuali non richiede né abilita inferenze simboliche fra bucket. Il conteggio rimane interno al recognizer e conclude il test di cardinalità: MUST NOT diventare un operando inline di V, una nuova serie o un binding per un'altra costruzione.

#### Forme business che MUST rimanere distinguibili

| Frase | Test del periodo | Quantificazione |
|---|---|---|
| Attivo ogni settimana | Almeno un'attività nella settimana | Tutte le settimane richieste |
| Attivo in almeno 3 delle ultime 4 settimane | Almeno un'attività nella settimana | Almeno 3 su 4 |
| Almeno 2 giorni attivi per settimana | Almeno 2 giorni distinti nella settimana | Tutte le settimane richieste |
| Almeno 2 giorni attivi in almeno 3 delle ultime 4 settimane complete | Almeno 2 giorni distinti nella settimana | Almeno 3 su 4 |

**Esempio informativo — O15.** Due giorni distinti con ascolto podcast in tre settimane e una quarta settimana vuota soddisfano la quarta forma. La settimana vuota resta nel denominatore di quattro; non è una settimana non osservabile e non viene eliminata.

---

## 15. Authoritative object contracts

### 15.1 Due fonti, non una relazione generale

**Correlated event object** significa che una identità riconoscibile negli eventi permette di correlare occurrence: product, order, message, project o ticket. Questo non prova che esista uno stato corrente autorevole dell'oggetto.

**Authoritative related state** significa che una fonte o una ricostruzione certificata dichiara la situazione corrente del singolo oggetto e la sua appartenenza al profilo. Il filtro può interrogarla senza diventare il responsabile di inventarne il lifecycle.

Lo stesso predicato di stato MUST avere lo stesso significato se alimentato da snapshot autorevole o da ricostruzione equivalente. Una ricostruzione accettabile richiede identità, stato iniziale o transizioni sufficienti, gestione delle rettifiche e regole di lifecycle esplicite. Non richiede necessariamente tutti gli eventi fin dall'origine quando un'evidenza successiva autorevole determina già lo stato; non basta però l'ultimo evento genericamente disponibile.

### 15.2 Authoritative Object Contracts ammessi

| Contratto ammesso | Evidenze richieste | Cosa può chiedere il filtro | Scorciatoie vietate |
|---|---|---|---|
| **Subscription corrente** | Appartenenza al profilo, piano e stato pagante attivo a T, validità/scadenza pertinenti | Esiste/non esiste una subscription pagante attiva; attributi del piano e periodicità dichiarati | Avvio storico = attivo; disdetta differita = cessazione immediata |
| **Ticket corrente** | Identità, appartenenza, stato attuale, apertura iniziale e lifecycle interpretati; scope Human Response se richiesto | Stato, età dall'apertura iniziale, quantificatori e test sullo scope Human Response dello stesso ticket | Aperti meno chiusi; risposta su un altro ticket; ultima riapertura al posto dell'apertura iniziale |
| **Cart corrente** | Identità, ultimo aggiornamento effettivo, completamento del medesimo carrello, contenuto corrente quando richiesto | Carrello non completato con ultimo aggiornamento nel periodo; requisito su uno dei suoi prodotti | Un vecchio aggiornamento qualificante; acquisto su altro cart; prodotti non più contenuti |
| **Trial corrente designato** | Un trial corrente univocamente designato per il contesto prodotto, identità, inizio, fine/stato e associazione dei comportamenti pertinenti | Stato/data del trial; attività nei suoi intervalli; assenze nel trial; attività dall'inizio | Ultimo TrialStarted assunto automaticamente come trial attivo; unire episodi di trial diversi |

Questi sono contratti mirati, non esempi di una capacità di navigazione arbitraria che l'utente può estendere autonomamente. Una nuova famiglia di oggetti con lifecycle proprio richiede una nuova estensione approvata del catalogo di contratti.

I corpi dei contratti non sono estensibili liberamente. Subscription espone stato e attributi correnti dichiarati; Ticket aggiunge lo scope nominato Human Response. Cart aggiunge il solo passaggio ai propri prodotti di §7. Il trial espone lo scope di Usage attribuito a quel trial e il solo slot temporale Upgrade dall'inizio fino a T sul repertorio di prodotto dichiarato, come precisato in §15.7. Nessun contratto espone qualsiasi comportamento soltanto perché possiede una identity o una data di inizio.

Un contratto Order MAY esporre stato corrente, totale, identity e lo scope nominato Refund. Questa dichiarazione non è necessaria alla riconciliazione event-based di I050, I076 e I077: la loro forma K correlata resta distinta dal consumo di uno stato Order autorevole. La presenza di order ID negli eventi non certifica automaticamente un Authoritative Object Contract Order.

Le informazioni del ciclo di fatturazione corrente — inizio, eventuale termine, quota e unità — costituiscono un insieme coerente di fatti personali autorevoli. Possono alimentare V e ancore personali senza richiedere un grafo di subscription.

### 15.3 Requisiti puntuali sui casi CORE

**I075.** La condizione è “esiste un ticket attualmente aperto”. Un altro ticket chiuso non lo compensa; una riapertura deve essere rispettata. Per “nessun ticket aperto”, occorre poter escludere ticket aperti anche tra eventuali oggetti o stati non ancora determinati.

**I074.** Il filtro verifica il vero ultimo aggiornamento del carrello prima di confrontarne la data. Non aggiunge automaticamente condizioni su valore, numero di articoli, scadenza commerciale o canale. “Non completato” riguarda quel carrello secondo il lifecycle dichiarato. Se il sistema riutilizza un identificatore per cicli di carrello diversi, il contratto MUST distinguere le identità di lifecycle: non può cancellare il significato di completamenti precedenti riusando una chiave ambigua.

**S002.** I prodotti sono quelli contenuti nel carrello di riferimento nello stato pertinente a T, non un prodotto che vi è comparso in una qualunque versione storica. Per un carrello non completato, il contenuto deve essere coerente con il suo ultimo aggiornamento e con le eventuali successive rettifiche.

**S004.** “Durante quel trial” richiede appartenenza documentata al trial, attraverso identità esplicita oppure una regola di assegnazione temporale univoca certificata. Se esistono trial sovrapposti non distinguibili, il solo intervallo non basta. L'assenza di utilizzi negli ultimi sette giorni resta sul repertorio del prodotto richiesto, non automaticamente sul solo trial; “nessun upgrade dall'inizio” usa l'ancora temporale senza aggiungere una correlazione al trial non richiesta dal benchmark.

### 15.4 Fonte autorevole, inventario e freschezza

La fonte MUST distinguere un insieme corrente certamente vuoto da un insieme non disponibile. Un ticket aperto noto può essere sufficiente per MATCH anche se altri ticket sono incerti. Nessun ticket restituito da una fonte incompleta non basta per “nessun ticket aperto”.

Se due fonti discordano, il catalogo MUST avere una fonte di autorità o una regola di riconciliazione dichiarata. La baseline non applica silenziosamente “vince il record ricevuto per ultimo”. Se il conflitto resta rilevante e irrisolto, il risultato è UNKNOWN.

Una fotografia vecchia non è automaticamente una fotografia attuale. Il contratto deve dichiarare per quale momento è valido lo stato e con quale evidenza può essere considerato corrente. La mancanza di tale garanzia può rendere UNKNOWN una condizione corrente; un eventuale filtro “allo stato osservato ieri” sarebbe una domanda diversa.

### 15.5 Riconciliazione e insiemi assegnati

La riconciliazione dei rimborsi di I050 è supportata con misure locali sull'ordine e importi confrontabili; non richiede un dominio ordini navigabile. I076 e I077 usano correlazione di eventi, non un generico stato “ordine soddisfacente”.

**I097 è rinviato.** La baseline non offre un insieme autorevole di assegnazioni variabile per profilo collegabile liberamente a completamenti. L'estensione compatibile prevista è un **produttore mirato di completamento degli assegnati del percorso corrente**, non un RelationCondition generale. Il suo contratto dovrà preservare almeno un assegnato, identità materiale–percorso e copertura dell'insieme corrente. Un conteggio totale di completamenti non è un'approssimazione accettabile. S015, che dipende da questa capacità, resta quindi fuori dalla copertura completa della baseline.

### 15.6 Contract-defined named evidence scopes

Un **Authoritative Object Contract** può esporre proprietà/stato corrente certificati dell'oggetto, identity, lifecycle metadata ammessi e un numero finito di **named evidence scopes**. Gli ulteriori slot sono esclusivamente il passaggio a membri del Cart e gli slot temporali del Trial enumerati in §§7 e 15; non esiste un corpo comportamentale generale. Ogni versione del contratto MUST enumerare i nomi disponibili e gli slot consentiti; l'utente seleziona uno scope esistente e i suoi parametri ammessi, non ne definisce uno nuovo.

Ogni named evidence scope MUST dichiarare la semantica business, la grana delle occurrence, le proprietà tipizzate, i timestamp e l'eventuale ordine, gli orizzonti autorizzati, la relazione di appartenenza alla medesima object identity, le regole di completezza e le operazioni foglia ammesse. MUST essere automaticamente correlato all'identità del candidato che possiede il contratto.

La correlazione può essere certificata tramite l'identità registrata nelle occurrence o un'assegnazione equivalente prevista dal contratto. L'utente MUST NOT scegliere una arbitrary join key, un altro oggetto destinatario o una formula di associazione. Un'identità mancante in una singola occurrence è evidenza irrisolta, non autorizzazione a contare la risposta su qualunque oggetto.

Uno scope MAY consentire presenza, assenza, count, condizioni same-occurrence o altre riduzioni foglia già ammesse, **soltanto se il contratto le autorizza esplicitamente**. Lo scope MUST NOT aprire un nuovo contesto navigabile, contenere altri insiemi/percorso/pattern arbitrari o diventare una RelationCondition general purpose. Il numero finito di nomi non autorizza un numero illimitato di foglie in una condizione: resta il limite di §7.

#### Contratto Ticket / scope Human Response

Il contratto Ticket MUST poter rappresentare l'inventario corrente, l'appartenenza al profilo, l'identity del lifecycle del ticket, lo stato a T e **openedAt come apertura iniziale**, distinta da eventuali riaperture. Il named evidence scope **Human Response** MUST rappresentare risposte umane appartenenti a quel ticket, non risposte automatiche o risposte ad altri ticket. Il repertorio che distingue umano e automatico è dichiarato dal contratto.

Per la forma richiesta da O17, gli slot ammessi sono stato corrente, età dall'apertura iniziale e assenza/presenza/count delle risposte umane dalla medesima apertura fino a T; sono ammessi i normali predicati foglia sulle proprietà dichiarate delle risposte. Il periodo parte dall'apertura iniziale inclusa come istante di lifecycle e termina a T escluso; se il contratto espone l'ordine dell'occurrence iniziale, una risposta MUST appartenere a quel lifecycle e non precederne l'apertura effettiva. Non è necessario un campo precomputato “prima risposta”.

La costruzione **esiste ticket attualmente aperto, aperto inizialmente da più di 48 ore, senza alcuna risposta umana dalla sua apertura** MUST applicare tutti i requisiti al medesimo ticket. “Più di 48 ore” è strettamente maggiore di 48 ore; una riapertura recente non azzera l'età iniziale. Una risposta durante una precedente fase dello stesso lifecycle impedisce di dire “nessuna risposta dalla sua apertura”; una risposta su un'altra identity no. Un ID riutilizzato per nuovi ticket deve essere distinto dal contratto.

Il filtro MUST NOT introdurre una capacità speciale denominata “Ticket Response”. Questa è un'applicazione di K con stato autorevole e un named evidence scope, non una nuova primitiva. Il contratto strutturalmente non disponibile rende la forma non attivabile; un inventario o uno scope incompleto per il singolo profilo può produrre UNKNOWN. Un testimone certo può comunque determinare MATCH secondo §19.

#### Contratto Order / scope Refund

Quando un contratto Order pubblica lo scope Refund, le occurrence MUST appartenere a quell'ordine e avere grana dichiarata, per esempio operazione di rimborso. Presenza di Refund, numero di operazioni e somma degli importi sono test distinti e disponibili soltanto se autorizzati. La somma dei rimborsi dello stesso ordine MAY essere confrontata con il totale certificato mediante V; lo scope non consente di passare a pagamenti, clienti o altri ordini arbitrari.

### 15.7 Slot temporali di contratto e correlazione d'identità

I named evidence scopes sono sempre object-correlated. Il **solo slot Upgrade del trial** è invece una foglia di comportamento del profilo ancorata all'inizio del trial corrente sul repertorio di prodotto dichiarato. MUST NOT essere denominato named evidence scope del trial, né presentato come prova che l'Upgrade appartenga a quel trial. Non richiede un trial ID sugli Upgrade e non autorizza altri comportamenti scelti liberamente.

Lo scope Usage “durante quel trial” è invece correlato al trial, tramite identity o assegnazione univoca certificata. La differenza fra questi slot MUST restare nel contratto, nella condizione salvata e nel riepilogo. Cart → suo prodotto resta il distinto passaggio a membri già autorizzato; aggiungere named evidence scopes non estende tale passaggio.

La possibilità di consumare questi contratti è una capability del linguaggio. Certificare che una fonte concreta li soddisfi è una dipendenza di dominio: MUST NOT essere dedotto dalla sola presenza di eventi raw con nomi simili.

---

## 16. Certified derived evidence

### 16.1 Definizione e responsabilità

**Certified Derived Evidence** è un'evidenza prodotta da un domain adapter, semantic producer o risultato analytics pubblicato, dotata di significato e adeguatezza certificati per gli usi autorizzati. È una capacità **consumabile dal filtering**, non una sesta primitiva e non un linguaggio per configurare la computazione esterna.

Il producer può pubblicare uno scalare, una selected/certified occurrence, uno stato o una milestone. Quando pertinenti, il contratto MUST includere subject/object identity, semantic value/state, occurrence o milestone timestamp, intervallo di validità, parametri di periodo e soglia, unità/grana e observation/completeness metadata. Gli ulteriori metadati richiesti dal dominio MUST essere esplicitamente certificati, non ricostruiti dal filtro.

Il contratto MUST dichiarare quali primitive e slot possono consumare ogni forma. Un valore scalare non diventa automaticamente occurrence; uno stato corrente non diventa una storia; un timestamp non rende disponibili le altre proprietà dell'evento originario. La pubblicazione non recupera testimoni lessicali di altre condizioni.

Il filtro MUST NOT conoscere o ricostruire l'algoritmo domain-specific che ha prodotto l'evidenza. Può applicare soltanto i normali test e contesti ammessi, ai dati certificati effettivamente esposti. Il fatto che un producer sia parametrizzabile non autorizza script, funzioni arbitrarie o chiamate composizionali a producer nel filter language.

### 16.2 Consumo tramite le cinque primitive

| Evidenza certificata | Consumo ammesso | Limite |
|---|---|---|
| Scalare tipizzato corrente o locale ammesso | Confronto tipizzato negli slot autorizzati | Non rende inline l'algoritmo esterno e non restituisce il testimone |
| Occurrence o milestone con identity e tempo adeguati | Behavior Scope, E o passo di P, nei ruoli dichiarati | Non è un evento sostitutivo generico di qualunque fenomeno |
| Stato corrente certificato | Proprietà ammesse o K tramite un Authoritative Object Contract | Non abilita navigazione o lookup storico |
| Proprietà storica certificata di una occurrence/milestone | Normale predicato sulla medesima evidenza | Non viene ricalcolata dallo stato attuale |
| Episodio temporale ammesso | Soltanto gli slot previsti dal recognizer o contratto | Non abilita concatenazione libera di pattern |

Un producer strutturalmente assente rende non attivabile la costruzione che lo richiede. Un producer presente con evidenza incompleta, non valida per il momento e l'uso richiesti o insufficientemente precisa per una selezione può produrre UNKNOWN, con ragione distinguibile. La validità di uno stato corrente e la validità di una milestone storica sono valutate secondo i rispettivi contratti: una milestone passata non scade soltanto perché è passata. Un fallimento tecnico di produzione o lettura non viene trasformato in UNKNOWN semantico per nasconderlo (§19).

### 16.3 Applicazione di dominio: video

Questa sottosezione specifica l'interfaccia di evidenza necessaria ai casi video già presenti, **non un algoritmo delle primitive**. Un adapter certificato può pubblicare copertura unica del contenuto nel periodo richiesto, milestone del raggiungimento del 90%, timestamp del primo raggiungimento pertinente e disponibilità dell'episodio successivo nel contesto e al momento rilevanti.

Per soddisfare I051 e S012, quando questi casi sono abilitati, il contratto del producer MUST distinguere identità/versione del contenuto, durata pertinente positiva, periodo W, soglia q e copertura unica riferita a W. Le rivisioni della stessa porzione non aumentano la copertura unica, anche se possono aumentare il tempo effettivamente fruito. L'esattezza di tale significato è responsabilità del producer; il filtro non legge porzioni o segment ranges.

Per S012 il producer MUST fornire il **primo raggiungimento del 90% cumulato a partire dall'inizio di W**, non una qualunque completion o l'ultima rivisione. La milestone deve conservare l'identità di episodio/serie, il periodo e il momento pertinenti. Una completion lifetime non è sostitutiva se non ne è certificata l'equivalenza con la domanda nel periodo.

Il fatto che l'episodio successivo fosse disponibile alla milestone del secondo episodio MUST essere una proprietà storica già certificata dell'evidenza pertinente, tenendo conto del destinatario quando necessario. Disponibilità oggi non prova disponibilità allora. L'assenza di questa proprietà certificata non autorizza il percorso a navigare verso un contenuto o a fare un lookup storico.

Le primitive MUST NOT conoscere segment ranges video, implementare l'unione delle porzioni, contenere slot chiamati “video” o acquisire funzioni storiche video-specifiche. Il catalogo business MAY avere una recipe video; la recipe configura soltanto il consumo dell'evidenza.

### 16.4 Language capability e domain capability

**LANGUAGE CAPABILITY:** il linguaggio MUST saper consumare certified derived evidence nelle forme e negli slot autorizzati.

**DOMAIN CAPABILITY:** I051 e la parte video di S012 sono supportati **quando** è disponibile il producer/adapter certificato che soddisfa il contratto pertinente. Questa versione MUST NOT dichiarare che il core universale li supporti autonomamente o che debba includere obbligatoriamente quel producer.

Un solo valore “copertura ≥90%” può essere sufficiente a uno specifico confronto, ma non prova da solo il primo timestamp necessario a un percorso. Certificare una soglia già raggiunta non certifica automaticamente l'assenza di raggiungimenti precedenti. Le diverse qualità di evidenza MUST essere propagate secondo il requisito effettivamente consumato.

Altri risultati esterni, come propensity score corrente o ranking già pubblicato con platea e parità dichiarate, possono essere consumati come dati certificati. La modellazione predittiva e il ranking dinamico rimangono esterni al filter language. Nessun credito di copertura nativa deriva dal solo fatto di poter leggere il risultato.

---

## 17. Time model and anchors

### 17.1 Istanti, durata e calendario

T è unico per l'intero filtro. La membership è valutata su eventi avvenuti prima di T. Eventi con timestamp futuro o uguale a T non sono usati come eventi già avvenuti. Le proprietà correnti sono interpretate come valide a T, non come proprietà storiche degli eventi.

Una **durata trascorsa** è distinta da un'unità di calendario. “Ultimi 30 giorni” significa trenta periodi di 24 ore; “ultime quattro settimane complete” significa quattro settimane del calendario scelto, da lunedì a domenica, esclusa la corrente. Il calendario predefinito per gli acceptance test è Europe/Rome, come nel benchmark riportato dalla Fase 3.

Il filtro MUST conservare il proprio calendario esplicito come riferimento predefinito. Una condition può dichiarare un calendario differente o il fuso corrente del profilo quando la richiesta lo prevede; tale scelta è locale alla condition, si applica al suo intero periodo ed è visibile. Il medesimo T resta comune anche quando i calendari differiscono. Un fuso corrente del profilo mancante o invalido non viene sostituito silenziosamente dal fuso del workspace. I058 usa proprio il fuso corrente del profilo, con fascia 18:00 inclusa–22:00 esclusa.

### 17.2 Set chiuso di ancore

| Ancora | Ammessa | Limite |
|---|---|---|
| T / ora della valutazione | MUST | Comune all'intero filtro |
| Intervallo assoluto | MUST | Estremi espliciti con timezone o istanti non ambigui |
| Periodo di calendario | MUST | Giorno/settimana/mese e intervalli relativi al calendario di T; nessuna formula arbitraria di calendario |
| Data personale corrente | MUST | Proprietà temporale tipizzata e dichiarata per questo uso, come iscrizione, creazione iniziale, inizio/fine ciclo o rinnovo |
| First / last / nth occurrence | MUST | Solo nel corpo della costruzione che l'ha selezionata |
| Passo di una sequence | MUST | Solo negli slot temporali del percorso: ritardo adiacente, durata totale, intervallo fra estremi, esito dal finale |
| Timestamp di lifecycle | MUST nei contratti ammessi | Trial corrente, aggiornamento del cart, ciclo o altra data espressamente pubblicata dal contratto |
| Ritorno qualificante | MUST | Solo nell'esperienza “ultimo ritorno qualificante” |
| Certified occurrence / milestone | Ammessa quando il contratto è disponibile | Solo nella costruzione E/P o nello slot autorizzato che la consuma; identity, tempo, periodo e validità devono essere certificati |

Sono ammessi offset costanti in unità di durata. Non sono ammesse espressioni che combinano timestamp arbitrari, medie di date, date prodotte da gruppi fratelli o calcoli del tipo “tre volte l'intervallo fra due eventi scelti liberamente”. Il preset I093 non espone i suoi intervalli come nuove ancore.

Una data senza orario può fungere da ancora soltanto mediante la convenzione dichiarata di inizio del giorno nel calendario esplicito. Date/ore locali inesistenti o ambigue richiedono disambiguazione; il builder non sceglie un istante nascosto.

### 17.3 Convenzioni degli estremi

| Espressione | Intervallo normativo |
|---|---|
| Ultimi Δ fino a T | [T−Δ, T) |
| Fra A e B giorni fa, A>B | [T−A giorni, T−B giorni) |
| Giorni/settimane completi | Dall'inizio del primo periodo all'inizio del periodo corrente, estremi [inizio,fine) |
| Date di calendario inclusive, per esempio 1–31 luglio | Dall'inizio del 1° luglio all'inizio del 1° agosto, [inizio,fine) |
| Nei Δ precedenti una data personale a | [a−Δ,a) |
| Nei Δ precedenti una occurrence a | Precedenza effettiva e ritardo ≤Δ; ancora esclusa; sono ammesse parità di timestamp ordinate affidabilmente |
| Entro Δ dall'iscrizione o da una data personale | Da a ad a+Δ, limite massimo incluso; ritardo non negativo |
| Nei primi Δ del trial | [a,a+Δ), oltre all'appartenenza al trial quando richiesta |
| Nei Δ successivi a una occurrence | Occurrence effettivamente successive con ritardo ≤Δ; limite massimo incluso |
| Dopo una occurrence fino a T | Precedenza effettiva rispetto all'ancora e timestamp < T |
| Prossimi Δ per una proprietà di data, come rinnovo | [T,T+Δ) |
| Ciclo corrente finora | Dall'inizio del ciclo a T, [inizio,T); non l'intero ciclo futuro |

Per I096 le due finestre sono [upgrade−14 giorni, upgrade) e le occurrence effettivamente successive all'upgrade con ritardo ≤14 giorni. L'upgrade selezionato fra 90 e 14 giorni fa rende il periodo successivo maturato secondo gli estremi sopra indicati.

Quando a è una occurrence, il bordo su a nelle finestre “prima” e “dopo” incorpora la sua posizione nell'ordine affidabile, non il solo timestamp. “Nei Δ prima” ammette una occurrence che precede a con ritardo non superiore a Δ, anche con ritardo zero se l'ordine lo prova; “nei Δ dopo” usa la regola speculare. La occurrence ancora stessa è esclusa da entrambe. Per una semplice data personale non esiste questa informazione di ordine: vale il normale intervallo di istanti dichiarato. Gli intervalli assoluti e quelli terminanti a T conservano sempre i loro estremi temporali [inizio,fine).

### 17.4 Finestre non mature e ancore mancanti

Una finestra che si estende oltre T non viene accorciata silenziosamente. Le prove positive già ottenute possono bastare per richieste monotone; un'assenza sull'intera finestra futura non è dimostrata. La sua condizione resta UNKNOWN quando l'esito dipende dalla parte non ancora trascorsa.

Un'ancora personale sconosciuta produce UNKNOWN per la richiesta dipendente. Una esperienza richiesta ma certamente inesistente produce invece NO MATCH per la costruzione che la richiede. Una data corrente incoerente con il proprio contratto, come un ciclo dichiarato corrente ma non interpretabile come tale, è evidenza invalida, non autorizzazione a scegliere un ciclo diverso.

Il prodotto MUST distinguere il periodo di selezione dell'ancora, il periodo dell'attività e l'eventuale deadline. Un unico controllo “timeframe” che li sostituisce tutti non è sufficiente.

---

## 18. Current vs historical state

### 18.1 Quattro significati non intercambiabili

| Fonte | Significato normativo | Inferenza vietata |
|---|---|---|
| Current profile state | Proprietà del profilo valide a T | Usarla implicitamente come proprietà del profilo a un evento passato |
| Event-captured property | Valore registrato sulla specifica occurrence | Assumere che rappresenti lo stato storico del profilo senza semantica certificata |
| Certified historical state | Stato relativo a un istante/intervallo passato, certificato dal contratto | Ricostruirlo mediante un lookup generale nel filtro |
| Current authoritative object state | Stato dell'oggetto appartenente al profilo, certificato per T | Deducerlo dalla sola presenza di un evento storico o da un saldo di eventi |

Current state MUST NOT essere usato implicitamente come historical state. La semantica del mapping MUST indicare a quale soggetto e a quale tempo si riferisce il campo: un nome uguale non rende i significati equivalenti.

**Esempi informativi.** “profile.plan = Pro” riguarda il piano attuale. “Purchase.plan = Pro” riguarda il valore catturato nel Purchase. “Il piano era Pro quando avvenne Purchase” richiede che quel campo o un'evidenza mirata certifichino proprio lo stato del profilo a quel momento. “Purchase.plan differisce dal piano attuale” è un confronto evento–profilo corrente esplicitamente ammesso e non richiede di ricostruire il passato.

### 18.2 Storia certificata e dati non disponibili

Una proprietà storica già acquisita e correttamente certificata può essere filtrata come dato. Una cronologia separata non autorizza il filter builder a ricostruire il valore di qualsiasi proprietà al tempo di qualsiasi evento. Un producer mirato MAY pubblicare il risultato, ma è una dipendenza distinta dalla capacità di consumarlo.

Se la capability strutturale necessaria non esiste, la costruzione è non attivabile o deferred secondo il contratto. Se esiste ma l'evidenza storica pertinente manca per il profilo, il risultato dipendente può essere UNKNOWN. MUST NOT essere usato il dato corrente come fallback.

### 18.3 Autorizzazione corrente e storica

I098 richiede autorizzazione pertinente **al momento dell'invio**. I100 riguarda il diniego o la revoca **correnti** del canale usato dall'invio. Un invio autorizzato allora e seguito da revoca non soddisfa per questo la domanda storica di I098; può soddisfare la domanda corrente di I100.

La revisione 1.1 proposta può consumare lo snapshot storico già certificato, ma non include il producer che lo ricostruisce da una cronologia separata: la generalità di I098 rimane deferred. La selezione di una fonte o la certificazione di un dato non ridefiniscono questo confine.

Lo stesso principio vale per la disponibilità storica certificata di un contenuto in §16: è un attributo dell'evidenza esterna, non una funzione storica aggiunta alle primitive.

---

## 19. Null / missing / insufficient evidence

### 19.1 I tre risultati

| Risultato | Significato | Membership |
|---|---|---|
| **MATCH** | La richiesta è dimostrata dalle evidenze pertinenti | Include il profilo |
| **NO MATCH** | È dimostrato che almeno un requisito necessario della richiesta non è soddisfatto, secondo la composizione prevista | Esclude il profilo, con ragione negativa certa |
| **UNKNOWN** | Le evidenze o l'osservazione non consentono di determinare il risultato necessario | Esclude il profilo dal segmento, ma non lo presenta come non corrispondente con certezza |

UNKNOWN comprende almeno: valore ignoto, appartenenza allo scope incerta, storia incompleta, ordinamento/identità insufficienti, finestra pendente, ancora ignota, stato corrente non certificabile o evidenza specializzata insufficiente. Il normale filtro MUST NOT offrire un cambio implicito a “includi anche gli UNKNOWN”; un'eventuale vista diagnostica degli ignoti è distinta dal segmento corrispondente alla richiesta. Queste ragioni MUST rimanere distinguibili nella diagnostica.

Una configurazione semanticamente invalida non produce UNKNOWN per ogni profilo: **non è valutabile come filtro valido**. Un errore o fallimento di esecuzione è un risultato non disponibile, non NO MATCH e non un UNKNOWN semantico inventato per nascondere il fallimento.

La rappresentazione SQL prevista per una valutazione ternaria è una espressione booleana `TRUE` / `FALSE` / `NULL`; la query che la utilizza decide selezione o aggregazione. L’implementazione esistente di `Applies` in Go mantiene invece `bool` per l’applicabilità del singolo record: `true` soltanto quando la condizione è dimostrata. Il booleano Go non costituisce una diagnostica ternaria. Nei body comportamentali, soprattutto con `None`, `All`, cardinalità e assenze, il motore non può scartare gli eventi con predicato SQL `NULL` per dedurre una prova di assenza.

### 19.2 Assenza di proprietà, null e vuoto

Per una fotografia corrente attendibile la distinzione dipende anche dalla **rappresentazione della proprietà**. Le proprietà ordinarie dei profili sono ReadOptional e **non Nullable**: nella tabella Profiles, SQL `NULL` significa proprietà senza valore. Nei percorsi JSON, invece, JSON `null` è un valore presente, diverso dal percorso assente.

| Stato della proprietà | `exists` | `does not exist` | `is null` | `is empty` | `is 100` / `is not 100` |
|---|---|---|---|---|---|
| Proprietà ordinaria Profiles assente (SQL `NULL`) | FALSE | TRUE | Non ammesso su quel tipo di schema | Secondo il tipo, se l'operatore è ammesso | SQL NULL / SQL NULL |
| Percorso JSON assente | FALSE | TRUE | FALSE | TRUE | SQL NULL / SQL NULL |
| Percorso JSON con JSON `null` | TRUE | FALSE | TRUE | TRUE | FALSE / TRUE |
| Percorso JSON con Number `100` | TRUE | FALSE | FALSE | FALSE | TRUE / FALSE |
| Percorso JSON con String `"100"` | TRUE | FALSE | FALSE | FALSE | FALSE / TRUE |

Per una proprietà ordinaria non Nullable, `exists` e `does not exist` rappresentano rispettivamente il valore presente e il valore mancante. Per una proprietà JSON il controllo di presenza distingue una chiave assente da un JSON `null` presente. Gli altri operatori di stato, compreso `is not null` e `is not empty`, seguono §8.2 e non sono ridefiniti come l'opposto del risultato di un confronto qualunque. “Non osservabile” a causa di evidenza incompleta differisce da “certamente assente”: può produrre UNKNOWN anche per test di presenza in scope che richiedono attestazioni di copertura.

Un operando staticamente incompatibile con lo schema o non consentito dall'operatore è una **configurazione invalida**. Un valore JSON dinamico lecito, ma di kind non applicabile ad un'operazione ordinata/testuale/contenimento, produce UNKNOWN per quella valutazione. Un dato ordinario che viola il proprio schema è evidenza invalida da trattare secondo il contratto del contesto, non una conversione implicita. Nessuna conversione di comodo è ammessa.

`currency != EUR` con currency mancante è UNKNOWN. Un requisito **esplicito** “currency è nota AND currency != EUR” può essere NO MATCH per quella occurrence perché la prima parte è certamente falsa. Questo è un test sullo stato del dato deliberatamente aggiunto, non la regola implicita di ogni disuguaglianza. I002 e I020 richiedono esplicitamente un valore noto; I019 e I021 non possono scartare valute ignote per dimostrare un'assenza o un “tutti”.

### 19.3 Evidence-aware scopes

Il motore semantico distingue occurrence certamente nello scope, certamente fuori e possibilmente nello scope. Per esempio, un Purchase con valuta ignota è possibile evidenza di un acquisto EUR. Può impedire di provare I019, pur non costituendo una prova positiva di acquisto EUR.

Una prova non è invalidata da dati ignoti irrilevanti: un acquisto EUR certo confuta già “nessun acquisto EUR”; tre acquisti certi provano già “almeno tre”. La stessa evidenza non prova “esattamente tre” se ne sono possibili altri.

### 19.4 Regole quantitative sull'incertezza

| Operazione | Regola normativa di evidenza |
|---|---|
| Count | Usa almeno il numero di qualificanti certi e al massimo quello dei qualificanti possibili, includendo occurrence non osservate dove la copertura è incompleta. Se non esiste un limite superiore certificato, il conteggio può crescere senza un massimo noto |
| Distinct | I valori certi costituiscono un limite inferiore. Valori ignoti o nuove occurrence possono aumentarlo; valori già noti e identici non lo aumentano. La stessa chiave non è contata due volte |
| Sum | Somma esatta solo quando popolazione e importi pertinenti sono determinati. Con valori certificati non negativi, una somma parziale può fornire un limite inferiore; con valori di segno non limitato non prova una soglia superiore al solo totale osservato |
| Average | Richiede popolazione e valori adeguati per quella media; la baseline non calcola una media “dei valori non-null”. Una guardia minima certamente falsa può comunque confutare la condizione |
| Min/max | I valori osservati possono fornire prove unilaterali: un max certamente >100 prova quella soglia, ma non prova max ≤100. L'insieme vuoto non produce un valore sentinella |
| Share per occurrence | Con D candidati noti, S successi certi e U esiti incerti, la quota è compresa fra S/D e (S+U)/D. Gli incerti restano in D |
| Ratio/differenza/confronto | Conservano i vincoli degli operandi e le guardie. Non si sostituisce un valore ignoto con zero. Se i valori possibili attraversano la soglia decisionale, il risultato è UNKNOWN |
| Prevalenza/quota massima | Le categorie ignote rimangono massa non assegnata, non una categoria fittizia. Un vincitore o limite si dichiara certo solo se l'incertezza non può modificarne la verità |

Un confronto è MATCH se i valori compatibili con l'evidenza ammessa lo soddisfano certamente, NO MATCH se lo violano certamente, UNKNOWN negli altri casi, salvo le guardie di validità esplicite della specifica misura. Non sono richieste inferenze simboliche tra condizioni sorelle: la composizione fra condizioni segue §20.

Non è ammessa una scelta d'implementazione “ignora i null nelle aggregazioni” come default. L'utente MAY restringere esplicitamente lo scope ai soli valori noti; il riepilogo deve allora dire che la popolazione è quella ristretta, non quella originale.

### 19.5 Completezza e maturazione

La completezza è relativa a **profilo, repertorio, grana, proprietà necessarie e intervallo**. Retention nominale e presenza di alcuni eventi non sono certificati di completezza. Anche l'identità del profilo e la deduplicazione possono limitare l'evidenza.

Per “mai” occorre poter osservare l'intera relazione rilevante. Per “primo di sempre” occorre escludere predecessori pertinenti. Per “ultimo” può bastare una coda storica completa dalla occurrence candidata a T. Per l'assenza di risposta entro una deadline occorrono maturazione e copertura della finestra, compreso l'estremo massimo incluso.

Una certificazione di osservazione è parte dell'evidenza corrente; non promette che non arriveranno mai correzioni. Una rettifica o un evento tardivo con tempo d'azione pertinente MUST poter cambiare la valutazione successiva e la sua spiegazione.

### 19.6 Warning e divieti di attivazione

La UI MUST mostrare un warning quando una costruzione valida produce UNKNOWN per parte dei profili a causa di storia, chiavi, proprietà, ordine, maturazione o stato insufficienti. MUST distinguere tali esclusioni dalle esclusioni certe e mostrare l'intervallo richiesto rispetto a quello adeguatamente osservato.

Una condition MUST essere **non attivabile** quando manca una capacità strutturale indispensabile: ruolo di identità per “stesso oggetto”; fonte di stato per ticket correnti; producer di certified derived evidence richiesto dalla domanda; nozione documentata di inizio/copertura per “mai”; repertorio osservabile per “prima azione successiva”. Non viene sostituita da una approssimazione.

Se il requisito strutturale esiste ma è soddisfatto soltanto per parte dei profili, la condition è ammessa con UNKNOWN per gli altri. Per esempio, una retention limitata può comunque coprire l'intera relazione di profili recenti. Una condition negativa non viene vietata soltanto perché alcuni dei suoi risultati saranno ignoti.

### 19.7 Ragioni diagnostiche senza nuovi truth values

UNKNOWN rimane un unico valore logico. La spiegazione MUST distinguere almeno **missing information**, **incomplete history**, **pending deadline**, **uncertain ordering** e **uncertified object state**; può inoltre indicare identity incerta, ancora ignota, appartenenza incerta allo scope o derived evidence insufficiente. Più ragioni possono coesistere senza diventare nuovi truth values.

| Situazione | Trattamento |
|---|---|
| Costruzione valida, evidenza pertinente non sufficiente | UNKNOWN semantico, salvo una prova decisiva sufficiente |
| Forma fuori dal linguaggio o parametri staticamente incompatibili | Invalid configuration; nessuna normale valutazione di membership |
| Mapping, contratto o producer strutturalmente indispensabile assente | Capability non attivabile; nessuna approssimazione |
| Esecuzione tecnica non riuscita o risultato non disponibile | Execution failure; non è NO MATCH né UNKNOWN semantico inventato |

Le regole quantitative di questa sezione applicano i vincoli dichiarati dagli operandi, dalle guardie e dall'evidenza, non scelte discrezionali di ottimizzazione. Le operazioni e i pattern chiusi possono fornire prove unilaterali espressamente ammesse; un valore ignoto non viene sostituito da una stima.

### 19.8 Membership corrente e rettifiche

La membership MUST essere determinata dal risultato corrente a T. **MATCH → UNKNOWN implica che il profilo non è più membro del risultato corrente.** MUST NOT essere applicata una semantica “last known match”, nemmeno per evitare oscillazioni operative. UNKNOWN → MATCH può includere il profilo; MATCH → NO MATCH lo esclude.

La perdita di una certificazione, una rettifica dei dati o il cambiamento del tempo possono modificare il risultato. Un risultato passato MAY essere conservato come evidenza storica con il suo T, ma MUST NOT sostituire quello corrente. In caso di execution failure, un vecchio risultato non deve essere dichiarato come nuova membership corrente verificata.

---

## 20. Boolean composition

Il filtro principale combina risultati di profilo mediante **AND** e **OR**, con associatività e commutatività logiche e i limiti di nesting di §7. Un filtro completamente privo di condizioni include l'intera platea Profiles valutata; non è un gruppo vuoto nascosto. Gruppi AND/OR esplicitamente vuoti non sono configurazioni valide. Un gruppo con una sola condizione ha il significato di quella condizione. La valutazione segue la logica ternaria forte:

| A | B | A AND B | A OR B |
|---|---|---|---|
| MATCH | MATCH | MATCH | MATCH |
| MATCH | NO MATCH | NO MATCH | MATCH |
| MATCH | UNKNOWN | UNKNOWN | MATCH |
| NO MATCH | NO MATCH | NO MATCH | NO MATCH |
| NO MATCH | UNKNOWN | NO MATCH | UNKNOWN |
| UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN |

Le righe simmetriche hanno il medesimo risultato. Un requisito certamente falso basta a confutare una congiunzione; un'alternativa certamente vera basta a provare una disgiunzione. Non è richiesto che tutte le condizioni di un profilo siano determinabili per includerlo o escluderlo correttamente.

**Non è previsto NOT generico**, né sul gruppo né su un filtro salvato. Gli operatori negativi dei 26 operatori pubblici sono condizioni autonome: non si riscrivono obbligatoriamente in una negazione di gruppo. L’utente dispone inoltre delle negazioni tipizzate richieste dal prodotto: diversità scalare, proprietà assente, zero occurrence, nessun candidato/opportunità che soddisfa un corpo, nessuna risposta nell'orizzonte e stato corrente esplicitamente negativo. Le coppie risposta/mancata risposta in un dato orizzonte preservano UNKNOWN; non fanno diventare vero un esito pendente.

Queste forme non sono intercambiabili. “Non ha acquistato EUR” non è “ha acquistato non-EUR”; “nessuna opportunità risolta” può essere vero senza opportunità, mentre “tutte le opportunità non risolte” richiede un insieme non vuoto.

Una macro MAY combinare più condizioni con AND/OR, ma MUST mantenere i confini dei contesti interni. Espandere una recipe non consente di spostare fuori da un contesto una foglia che usa la sua ancora o la sua chiave. Una modifica del genere è invalida; non rende quella foglia magicamente indipendente.

### 20.1 Distinzioni negative da preservare

| Forma | Oggetto della negazione tipizzata | Cosa non significa |
|---|---|---|
| Negative scalar operator | Un valore noto, ad esempio diverso da EUR | Assenza di occurrence; dato mancante automaticamente qualificante |
| Zero qualifying occurrences | Nessuna occurrence qualificante nello scope osservato | Presenza di una occurrence con proprietà opposta |
| No matching candidate | Nessun candidato soddisfa il corpo autorizzato | Tutti i candidati soddisfano il corpo opposto, con non-vuoto implicito |
| No outcome for an opportunity | Esito assente per quella opportunità nell'orizzonte | Nessun esito per qualunque altra opportunità |
| Negative step / deadline | Assenza locale nell'intervallo o nell'orizzonte del percorso | Automazione in attesa o complement di un percorso qualsiasi |
| Complement of arbitrary expression | Non ammesso nella revisione 1.1 proposta | Non viene introdotto mediante un alias o una recipe |

Gli operatori negativi preservano UNKNOWN quando l'evidenza non consente la conclusione; non trattano l'assenza di prova come prova dell'assenza.

---

## 21. Recipes

Una recipe è una definizione di prodotto riconoscibile che configura capacità semantiche esistenti. **Non è un linguaggio di estensione né un nuovo algoritmo nascosto.**

| Forma | Esempio | Responsabilità |
|---|---|---|
| Shortcut | “Non ha mai acquistato”; “valore medio ordine” | Presentazione business di una capacità generica; conserva periodo, grana e guardie |
| Macro | “Prima attivo, ora inattivo”; “ha abbandonato Export ma usa il resto” | Composizione ispezionabile di condizioni standard |
| Semantic/domain recipe | “Spesa netta”; “carrello con interesse ripetuto”; “quota del ciclo corrente” | Configurazione che richiede mappings o contratti di dominio espressamente supportati |

### 21.1 Definizione e modifiche

Una recipe salvata MUST conservare il significato della definizione scelta e un riferimento riconoscibile alla propria origine. Aggiornare la recipe del catalogo MUST NOT cambiare silenziosamente il significato di filtri già salvati. Una nuova definizione può essere adottata mediante scelta esplicita con differenze leggibili.

L'utente MUST poter aprire la logica sottostante. Cambiare i parametri previsti — periodo, soglia, comportamento, valuta, minimo — mantiene la recipe finché ne conserva gli invarianti. Cambiare la struttura logica crea una **condizione personalizzata derivata dalla recipe**: resta la provenienza, ma l'etichetta non deve più promettere la definizione originaria.

Per una recipe di dominio, disattivare una guardia semantica indispensabile non è una semplice personalizzazione lecita. Per esempio, una media monetaria senza importi confrontabili o una risposta senza identificazione dell'opportunità non possono conservare il nome business.

### 21.2 Espansione e leggibilità

La vista espansa MUST mostrare base, selezione, requisiti, quantificatore, periodo dell'esito e guardie. La vista chiusa MUST conservare le differenze decisive; “ha abbandonato” da solo non basta a descrivere carrello, ultimo aggiornamento e completamento.

Una macro può apparire come una sola card, ma il suo significato non dipende da questa scelta. I riferimenti locali rimangono nello stesso blocco anche nella vista espansa.

### 21.3 Contenimento del catalogo

Il catalogo SHOULD mantenere famiglie parametrizzate, non una recipe separata per ogni evento, periodo o soglia. “Spesa” con lordo/netto sono varianti distinte dello stesso accesso; “spesa negli ultimi 7/30/90 giorni” non sono tre nuovi concetti. I sinonimi migliorano la ricerca senza creare nuove definizioni.

Le recipe organizzative MAY essere pubblicate soltanto usando il linguaggio ammesso e dichiarando proprietario, significato e prerequisiti. Pubblicare una frase non rende disponibile una capability che la baseline non supporta.

---

## 22. Semantic catalog

### 22.1 Decisione sul livello

Adottiamo **lightweight semantic mappings obbligatori per i concetti presentati come business, più un elenco chiuso di contratti di dominio mirati**. Lo schema raw resta utilizzabile per attività e proprietà tipizzate. Non adottiamo un rich data graph o una ontologia generale.

| Mapping | Informazione che MUST dichiarare |
|---|---|
| Ruolo dell'evento | Quali azioni costituiscono Purchase, Refund, Delivered, Clicked, Upgrade, Usage ecc.; quali esclusioni fanno parte del repertorio |
| Grana e identità dell'azione | Ordine, riga, operazione, messaggio, sessione di fruizione o altra evidenza; come evitare copie della stessa azione |
| Importo | Significato economico, segno, unità, grana, inclusione di tasse/spedizione/sconti e relazione con i rimborsi |
| Valuta / quantità / durata | Valori confrontabili, scala e unità; tempo fruito distinto da durata nominale |
| Entity key | Dominio, namespace e associazione alle occurrence; product, order, cart, project, ticket, message, campaign, feature, series |
| Repertori business | Utilizzo del prodotto, risposta, interazione marketing, azione utente; nessuna inclusione implicita basata sul solo nome |
| Tempo e copertura | Timestamp dell'azione, eventuale ordine affidabile, inizio della relazione e intervalli di osservazione pertinenti |
| Proprietà personali | Tipo e significato corrente; date ammesse come ancore; timezone quando usata |
| Contratti degli oggetti | Autorità, appartenenza, completezza dell’insieme, lifecycle, validità corrente e named evidence scopes ammessi |
| Certified derived evidence | Forma pubblicata, grana, periodo, identity, validità, completezza, tempi delle milestone e ruoli di consumo autorizzati |

Una associazione di ruoli non produce il dato mancante. Mappare product ID non ricostruisce i prodotti del carrello; mappare amount non definisce la spesa netta; chiamare un evento Completed non dimostra copertura video unica.

### 22.2 Policy corrente per canale

Contattabilità e autorizzazione corrente sono criteri distinti. La recipe di contattabilità legge la policy organizzativa dichiarata, comprensiva di contatto utilizzabile, autorizzazione ed esclusioni pertinenti.

I100 correla il canale dell'invio con l'autorizzazione corrente di **quel canale**, attraverso un mapping finito dei canali supportati. Può essere espresso come alternative esplicite per email, SMS, push ecc.; non richiede accesso dinamico a campi arbitrari. Solo diniego o revoca noti soddisfano “oggi non autorizzato”. Autorizzazione mancante o ignota non equivale a diniego.

### 22.3 Contratti mirati e definizioni certificate

Il catalogo dei domain contracts MUST dichiarare autorità dello stato, lifecycle metadata, appartenenza/inventario, named evidence scopes e certified derived evidence disponibili. Il mapping di un ruolo evento o di un campo non certifica da solo alcuna di queste proprietà.

I contratti degli oggetti sono disciplinati in §15; le evidenze derivate in §16; il confine fra corrente e storico in §18. Il catalogo MUST NOT duplicarne o modificarne le definizioni mediante alias business. Il semantic catalog MUST NOT diventare una copia completa del data model o un rich data graph.

Prima dell'attivazione di una voce business, significato, autorità e prerequisiti MUST essere dichiarati. Organizzazioni diverse possono usare mapping diversi, ma il filtro salvato MUST indicare la definizione pertinente. Una modifica del significato di mapping, repertorio, unità, grana, contratto o producer MUST NOT essere applicata silenziosamente a filtri già salvati.

I named evidence scopes MUST essere finiti per versione del contratto. L'aggiunta di un nome non ne rende accessibili gli interni e non abilita navigazioni, chiavi o operatori non autorizzati.

---

## 23. Inline vs derived

Una domanda è **inline** quando può essere parametrizzata mediante le costruzioni chiuse del filter language sui contesti ammessi. È **derived** quando richiede un modello, una ricostruzione di dominio o una computazione che non appartiene alle costruzioni chiuse. Costo, statefulness e riuso MUST NOT decidere da soli questa classificazione.

| Famiglia | Decisione normativa | Riuso possibile |
|---|---|---|
| Proprietà correnti, count, esistenza/assenza | MUST inline | MAY come criterio o valore nominato |
| Sum, avg, min/max, distinti, share, confronti chiusi | MUST inline | MAY essere pubblicati come valori derivati con la stessa definizione |
| Primo/ultimo/nth, ultime N, finestre personali | MUST inline | MAY pubblicare un valore scalare; non esportare il contesto di selezione |
| Ripetizione, prevalenza, novità e correlazione | MUST inline nelle forme enumerate | MAY pubblicare risultati; la pubblicazione non aggiunge traversali |
| Percorsi, opportunità ed esiti | MUST inline guidato | MAY riusare la definizione, conservando occasione, periodo e deadline |
| Pattern temporali e ritorni | MUST inline nelle forme enumerate | MAY pubblicare misure o episodi tipizzati; il loro uso resta locale |
| Stato di ticket/cart/subscription/trial | MUST filtrabile inline tramite contratti mirati | La produzione dello stato può essere autorevole o ricostruita; ciò non equivale a richiedere un nuovo trait per ogni filtro |
| Certified derived evidence, incluso il caso video | Consumo nelle primitive ammesse; produzione domain-specific esterna al core | I051 e S012 condizionali al producer pertinente; nessun indicatore scalare generico sostituisce identity, finestra o timestamp |
| Completamento degli assegnati, I097 | **Deferred**, candidato a produttore mirato | Non conta come supporto della baseline |
| Ricostruzione autorizzazione storica, I098 | **Deferred**, candidato a enrichment mirato | Una snapshot storica già certificata rimane filtrabile; non dimostra disponibilità del produttore generale |
| Mediana della cadenza personale, I094 | **Deferred**, principalmente derived | Nessun calcolo inline o sostituto approssimato nella baseline |
| Ranking di popolazione, I046 | Fuori dal linguaggio | Un risultato analitico esterno esplicito può essere letto come dato, senza credito di copertura nativa |

Un valore derivato MUST dichiarare unità, grana, popolazione osservata, periodo, ancora, guardie, adeguatezza dell'evidenza e momento di validità. Un generico engagement score non copre automaticamente diminuzione, interruzione o ritorno.

Pubblicare una misura non deve cambiarne il significato rispetto alla costruzione inline. La sua eventuale disponibilità in ritardo o con copertura differente MUST essere esplicita. Non è lecito riutilizzare una somma lifetime come somma degli ultimi trenta giorni perché entrambe sono etichettate “spesa”.

“Computed” non significa necessariamente proprietà piatta del profilo. Le milestone certificate devono preservare identità e tempo; gli episodi di ritorno devono preservare il ritorno pertinente. Viceversa, il semplice fatto che un motore precomputi una somma non la rende una capability primarily derived dal punto di vista del prodotto.

La pubblicazione di un valore o di uno stato può rendere consumabile il **risultato** di un calcolo esterno, ma MUST NOT far apparire quel calcolo costruibile inline. Un alias, una saved measure o una macro MUST NOT convertire un risultato terminale in operando: il consumo di un dato esterno richiede una pubblicazione certificata distinta, con propria definizione e validità, e non una composizione inline mascherata. Un risultato derived può essere scalare, occurrence certificata, stato o milestone; MUST NOT essere forzato a diventare un profile trait numerico che ne perda il significato.

La presenza di due risultati esterni atomici e certificati può consentire il normale confronto ammesso fra scalari; non consente di richiamare inline i producer, di comporne gli algoritmi o di recuperare i loro testimoni.

---

## 24. Filtering vs analytics vs automation

**Filtering** determina chi soddisfa ora la richiesta. Può richiedere aggregazioni, confronto con la situazione personale, ordinamento, correlazione e stato. Il fatto che un calcolo sia sofisticato non lo trasforma in analytics.

**Analytics** comprende formule libere, esplorazione statistica, ranking di platea, ricerca delle soglie e modelli non enumerati dal linguaggio. Un risultato analitico pubblicato come dato o segmento può essere consumato con una definizione e una validità esplicite, ma non rende automaticamente esprimibile il calcolo nel normale filter builder.

**Automation** decide quando reagire e che cosa fare: attese, invii, ingresso/rientro in un workflow, cancellazione di azioni pianificate ed esclusività di attribuzione. Questi aspetti non fanno parte della condition.

“Una email non è stata cliccata entro sette giorni” è filtering retrospettivo. “Invia un promemoria esattamente allo scadere dei sette giorni” è automation. Il trascorrere del tempo può cambiare membership anche senza nuovi eventi — scadenza, uscita da una finestra, cambio di stato previsto — ma la responsabilità di rivalutare il filtro non introduce azioni nel linguaggio.

Il filtro può controllare che una campagna sia già stata ricevuta dopo un ritorno, come S011. Non garantisce da solo che un'automazione concorrente non la invii due volte. Analogamente, una rettifica tardiva può correggere membership storica senza annullare automaticamente un'azione già compiuta da un sistema esterno.

---

## 25. Progressive disclosure and discoverability

### 25.1 Tre livelli, senza nascondere il significato

| Livello | Sempre comprensibile in quel livello | Esempio |
|---|---|---|
| **Richiesta comune** | Verbo/intento, oggetto o comportamento, periodo, misura/soglia, unità e quantificatore principali | “Ha speso più di €1.000 lordi negli ultimi 90 giorni” |
| **Precisione della richiesta** | Requisiti sulla stessa occurrence; base della selezione; minimo campione; peso della quota; contesto “stesso oggetto” quando scelto | “Valore medio >€100, con almeno due ordini” |
| **Configurazione specialistica** | Confronto fra due scope; nth/ultime N; esiti per opportunità; continuità; verifiche prima/dopo; corpi locali ammessi | “Dopo l'ultimo upgrade, utilizzi >2×prima, con almeno dieci prima” |

Sum, average e distinct MUST essere trovabili al primo livello di ricerca, non solo dentro una sezione tecnica “Advanced”. La complessità dell'operazione esecutiva non determina il suo livello di discoverability.

Una opzione può essere chiusa nella UI, ma la differenza che cambia membership MUST restare nel riepilogo: “primo del periodo” contro “primo di sempre”; “entro sette giorni” contro “fino a oggi”; “stesso prodotto” contro “qualsiasi prodotto”. Non si nascondono queste differenze sotto un badge generico “avanzato”.

### 25.2 Menu e ricerca

Il menu è organizzato principalmente per **intento**, con comportamento/oggetto come filtro contestuale. Le quattro aree di §5 orientano la navigazione; non sono l'unico accesso. Le recipe sono risultati del catalogo, non un secondo catalogo separato con centinaia di voci.

La ricerca MUST associare sinonimi e varianti linguistiche a definizioni esistenti: “speso/spent” a spesa; “ultimo/last” a selezione; “ripetuto/repeated” a ripetizione; “ritornato/returned” a ritorno; “preferito/preferred” a preferenza dichiarata o prevalenza osservata, mostrando la distinzione; “inattivo/inactive” ad assenza recente oppure interruzione di abitudine, senza fonderle.

Un risultato di ricerca deve identificare la richiesta, una frase d'esempio e gli eventuali prerequisiti. La ricerca semantica MAY proporre parametri, ma MUST NOT inventare operatori, scegliere implicitamente un significato ambiguo o cambiare una richiesta non supportata in una supportata diversa.

### 25.3 Riepilogo e spiegazione

La spiegazione MUST seguire la struttura della richiesta, non l'ordine del piano esecutivo. Per S002 mostra il carrello qualificante e il suo prodotto rivisto; per I048 le opportunità nel denominatore e il criterio di risposta; per I025 la base storica del primo acquisto.

Per una negazione non basta mostrare “0 risultati”: occorrono repertorio, intervallo, chiave e qualità dell'osservazione. Per UNKNOWN occorrono ragione e parte della richiesta non decidibile. Per una quota occorrono numeratore, denominatore, eventuali esiti incerti e minimo.

Le prove visuali e la formulazione delle card SHOULD essere validate con utenti target. L'esito di questi test può cambiare organizzazione e progressive disclosure; MUST NOT essere usato per semplificare una semantica corretta sostituendola con una diversa.

### 25.4 Leggibilità dei contesti, delle operazioni e dei prerequisiti

La UI MUST mostrare quali passi appartengono al blocco correlato e quali ne sono fuori. MUST NOT descrivere come “stesso ordine” un acquisto successivo esterno al blocco Purchase–Refund.

Nel corpo di un oggetto autorevole la UI MUST offrire soltanto i named evidence scopes del suo contratto e rendere leggibile “per questo ticket/ordine”; MUST NOT offrire un selettore di join key. Una card sul ticket può quindi esprimere stato, apertura iniziale e mancanza di risposte umane senza esporre relazioni tecniche.

Per l'attività periodica il riepilogo MUST distinguere il numero di periodi richiesti, il numero minimo di periodi qualificanti e il test in ciascun periodo. “Almeno due giorni in almeno tre delle ultime quattro settimane complete” non può essere abbreviato a “attivo settimanalmente”.

Un operatore terminale MUST NOT apparire fra i valori selezionabili per un'altra operazione. Le capacità condizionali MUST mostrare il contratto/producer richiesto e non promettere disponibilità universale. La visualizzazione MAY comprimere i dettagli, ma non occultare queste differenze di membership.

---

## 26. Supported / conditional / deferred capabilities

### 26.1 Perimetro delle capability

La disponibilità del linguaggio, del mapping, del contratto, dei dati per il profilo e dell'esecuzione sono livelli distinti. “Supportato” in questa tabella indica una forma semantica definita, non dati già disponibili in una CDP concreta.

| Famiglia | Stato nella revisione 1.1 proposta | Dipendenze e limite |
|---|---|---|
| Fatti correnti e scope eventi | Supportati nel core | Proprietà/eventi tipizzati, same-occurrence, periodi, grana/identity e osservazione pertinenti |
| Operandi e confronti | Supportati nel core | Riduzioni atomiche nominate; confronto diretto; difference, ratio, relative change e share terminali |
| Esperienze selezionate | Supportate nel core | First/last/nth, ultimi N, selezione prima della verifica e preset finiti; nessun fallback |
| Distribuzioni e identità osservate | Supportate nel core | Una dimensione per forma; distinti, ripetizione, prevalenza unica e novità; nessun GROUP BY libero |
| Correlazione event-based e opportunità | Supportate nel core con mappings pertinenti | Identity semantica, inizio, esito, orizzonte, quantificatori; non certificano stato corrente |
| Percorsi | Supportati nel core | 2–5 passi; al massimo una chiave e un blocco contiguo; ordine/durate/esiti chiusi |
| Activity patterns | Supportati nel core | Catalogo chiuso, inclusa qualified-period cardinality; nessuna formula di time series |
| Subscription, Ticket, Cart, Trial autorevoli | Consumo supportato; disponibilità condizionale | Contratti mirati di §15, inventario, lifecycle, stato e validità certificati |
| Named evidence scopes | Consumo supportato; disponibilità condizionale | Nomi e foglie dichiarati dal contratto; Human Response del Ticket; eventuale Refund dell'Order |
| Cart → propri prodotti → attività | Forma chiusa supportata con contratto Cart | Contenuto corrente certificato e product identity; nessuna traversale ulteriore |
| Certified derived evidence | Capacità di consumo supportata | Contratto del producer per forma, identity, tempo, validità e osservazione |
| I051 e parte video di S012 | Condizionali al producer video certificato | Non sono capacità autonome del core; nessun producer video obbligatorio implicito |
| Snapshot storico già certificato | Filtrabile come evidenza | Non equivale alla ricostruzione storica generalizzata |
| I097 / S015 | Deferred | Producer mirato per completamento degli assegnati non incluso; nessun costruttore libero di assegnazioni |
| Ricostruzione I098 da cronologia separata | Deferred | Snapshot storici corretti già disponibili rimangono consumabili |
| I094, cadenza personale statistica | Deferred | Futuro risultato derivato con definizione esatta, non editor statistico |
| I046, ranking dinamico di popolazione | Fuori dal linguaggio | Un ranking già pubblicato è un dato esterno, non supporto nativo al calcolo |
| O08, comportamenti di altri membri dell'account da relazioni/eventi raw | Fuori dal normale profile filter | Non viene aggiunto tramite join fra profili o named evidence scopes |

### 26.1.1 Ammissibilità operativa di `core.Filter`

| Ingresso | Capability massima ammessa |
|---|---|
| Query dei profili unificati tramite API; pagina Profiles nell’Admin | Record + Unified |
| Destination connections che esportano profili verso applicazioni, file e database | Record + Unified (selezione dei profili **prima** della trasformazione/esportazione) |
| Altre pipeline e richieste che già supportano `core.Filter` | Soltanto Record |
| Ingressi che già vietano `core.Filter` (per esempio Source/Database/User della baseline) | Restano vietati; nessuna ammissione introdotta |

La validazione server MUST applicare il limite di capability alla rule **prima dell'esecuzione** e anche per anteprime e filtri caricati: il nome Destination da solo non basta, conta l'esportazione di **profili**. La disponibilità di mapping/contratti rimane una verifica distinta dall'ammissibilità operativa. Consensi, autorizzazioni e altri vincoli dell'esportazione restano indipendenti dal filtro.

### 26.2 Core language coverage

Il core può esprimere le forme definite usando **Profile, Events, semantic mappings e le cinque primitive**, senza richiedere un producer verticale per il significato dell'operazione. Sono esempi di responsabilità semantica del core: same-occurrence, conteggi/somme/medie, selezione, confronto fra misure e profilo corrente, correlazione di order/product/message identity, percorsi con scoped correlation e pattern calendar finiti.

I050, I076 e I077 possono usare correlazione e importi negli eventi; la loro forma non richiede un oggetto Order autorevole. O03 usa il blocco correlato limitato ai primi due passi; O15 usa il riconoscimento per periodo e la cardinalità. Questi esempi non sono una nuova matrice o una riclassificazione di rilevanza.

Un valore esterno può essere leggibile mediante una forma core senza che il **calcolo del valore** riceva credito di core coverage. I prerequisiti di dati, identità, unità e osservazione rimangono vincolanti anche per una forma core.

### 26.3 Coverage with targeted domain contracts

Lo stato corrente di subscription, ticket, cart e trial, i loro slot autorizzati e i named evidence scopes richiedono contratti concreti conformi. La semplice disponibilità di Profile ed Events non certifica tali contratti.

I009, I074 e I075 richiamano stato autorevole; S002, S004, S011, S013 e S014 dipendono dai contratti necessari alle rispettive condizioni correnti. O17 richiede in particolare inventario Ticket, apertura iniziale e Human Response della medesima identity. I percorsi sugli eventi di chiusura/riapertura o le opportunità di risposta non vengono per questo fusi con lo stato corrente autorevole.

Un contratto soddisfatto da una ricostruzione certificata può essere semanticamente equivalente a uno snapshot autorevole. Il filtro consuma il contratto, non assume né implementa la ricostruzione.

### 26.4 Coverage with certified derived evidence

I051 conserva la classificazione COMMON richiamata dalla Fase 3, ma il suo supporto è condizionale al producer che certifichi la copertura unica pertinente. S012 richiede sia i contratti correnti delle altre condizioni sia milestone e disponibilità storica certificate. Le primitive possono consumare queste evidenze; non le producono autonomamente.

Analogamente, O12 può leggere un propensity score pubblicato e valido, e O16 una classifica già pubblicata; il modello predittivo e il ranking rimangono fuori dal linguaggio. I094, I097 e la ricostruzione I098 non diventano supportati dalla baseline solo perché un generico producer futuro sarebbe concepibile.

### 26.5 Regola di reporting

Il reporting MUST distinguere core language coverage, coverage con targeted domain contracts e coverage con certified derived evidence. Le dipendenze possono coesistere nello stesso segmento e MUST essere tutte riportate. MUST NOT essere utilizzata una percentuale unica che le nasconda o attribuisca al core un producer verticale.

Un conteggio aggregato MUST NOT essere presentato come copertura del core o garanzia incondizionata quando include contratti mirati o producer. Questo documento non assegna un nuovo score: il benchmark rimane acceptance evidence e conserva le proprie classificazioni, senza obbligare il linguaggio ad assorbire funzionalità verticali.

Il supporto semantico condizionale MUST essere distinto da capability deferred, prerequisito strutturale mancante, UNKNOWN sul profilo e fallimento di esecuzione. Un caso conditional non è un PASS operativo finché i suoi contratti e dati non sono disponibili.

---

## 27. Explicit exclusions

| Capacità esclusa | Confine preciso | Ragione di prodotto |
|---|---|---|
| Formule arbitrarie | Niente expression tree, codice, funzioni libere o composizione ricorsiva di misure | Le sole operazioni terminali nominate coprono i casi ammessi; il numero di operandi non valida da solo una formula |
| Join arbitrari | Niente condizioni di join per uguaglianza di campi qualsiasi, disuguaglianze, intervalli o espressioni | Una chiave e un solo blocco contiguo coprono la correlazione ammessa nei percorsi; chiavi diverse per archi diversi restano escluse |
| Variabili cross-condition | Nessuna occurrence/chiave/ancora può essere referenziata da un blocco fratello | Mantiene leggibilità e impedisce dipendenze nascoste fra card |
| Navigazione libera di relazioni | Nessuna catena oggetto→oggetto non esplicitamente ammessa | S002 giustifica un percorso membro specifico, non un intero data graph |
| Query e processi ricorsivi | Nessun loop, stato di workflow o percorso di lunghezza non limitata | I079 e I092 hanno costruzioni finite o episodi canonici; non richiedono un linguaggio universale di processi |
| Ranking/percentile di popolazione | I046 non è una condition dinamica nativa di Profiles | Cambia la membership in funzione degli altri profili e di una platea analitica; non è necessario ai CORE/COMMON |
| Nth-to-nth arbitrario e ordinamenti liberi | Niente confronto di posizioni arbitrarie; niente ordinamento per amount o score | I031 e I093 sono coperti con preset finiti; il resto aggiunge configurazione senza rendimento dimostrato dal benchmark prioritario |
| Funzioni statistiche generali | Niente mediana, deviazione, regressioni, quantili o baseline apprese nell'editor | La cadenza I094 può essere un futuro prodotto derivato, non motivo per un editor di statistiche |
| Time-series expressions arbitrarie | Niente formule fra bucket, stagionalità libera, trasformazioni concatenate o finestre parametrizzate da risultati di altre formule | I pattern enumerati mantengono spiegazioni operative precise |
| Stato storico generalizzato | Niente “valore di qualsiasi proprietà del profilo al tempo di qualsiasi evento” | Richiede un nuovo asse di evidenza temporale; snapshot certificati e produttori mirati non lo implicano |
| Correlazioni fra profili | Nessun comportamento congiunto di colleghi, familiari o altre persone nello stesso filtro | Il benchmark non definisce identità collettiva, quantificatori o ambito necessari; una proprietà aziendale non è un sostituto |
| Workflow/live triggers | Nessun “aspetta e invia”, consumo di match, rientro o esecuzione di azioni | Determinare membership e orchestrare azioni sono prodotti diversi |
| NOT generico | Nessuna negazione di interi gruppi o filtri salvati | Le negazioni tipizzate coprono i bisogni richiesti senza confondere assenza, diversità, non-vuoto e dato ignoto |
| Recuperi silenziosi del dato | Niente ultimo non-null, null→zero, storia disponibile→storia completa o stato corrente→storico | Non sono semplificazioni: cambiano la domanda e possono cambiare i destinatari |

Il filtro MAY leggere un dato esterno che descrive il risultato di un'analisi non nativa, purché il significato e la validità siano dichiarati. Questo non aggira il perimetro: il calcolo esterno non diventa costruibile inline e non riceve credito come capacità coperta.

Il comportamento degli altri membri dello stesso account MUST NOT entrare indirettamente mediante join fra profili, repertori estesi o named evidence scopes. Un futuro dominio account/related-entity esplicito oppure un derived value autorevole può fornire dati consumabili con un nuovo contratto approvato; questo non rende O08 rappresentabile sui soli eventi individuali e sulle relazioni raw disponibili nella revisione 1.1 proposta.

MUST NOT essere introdotti producer obbligatori di dominio nel core al solo scopo di aumentare il punteggio di coverage. MUST NOT essere esposti algoritmi di copertura video, lookup storico universale, trasformazioni dei bucket o traversali sotto etichette apparentemente chiuse.

---

## 28. Acceptance examples

### 28.1 Criterio del test

I seguenti sono **test semantici di accettazione**, non risultati di esecuzione su una CDP. Le fixture mantengono convenzioni e prerequisiti del benchmark come riportati dalla Fase 3; le integrazioni riprendono le decisioni finali e gli intenti out-of-sample forniti. Salvo indicazione diversa, identità, unità, ordine e storia richiesta sono adeguatamente osservati; T è comune.

Una implementazione successiva MUST produrre gli esiti indicati. Le righe “deferred” non sono promosse a PASS attraverso la disponibilità ipotetica di un trait custom.

### 28.2 Same occurrence e negativi

| Intento | Costruzione richiesta | Controesempio o esito atteso |
|---|---|---|
| **I022** | Un solo scope Fruizione con categoria Scienza AND dispositivo mobile, ultimi 30 giorni; esistenza | Scienza da desktop + Arte da mobile, senza altre fruizioni: **NO MATCH** |
| **I023** | Esistenza Fruizione Scienza AND esistenza Fruizione mobile, ciascuna in 30 giorni | Stessa fixture: **MATCH**, perché gli episodi possono essere diversi |
| **I019** | Zero Purchase con valuta EUR in 90 giorni | Zero acquisti: **MATCH**. EUR+USD: **NO MATCH**. Solo un acquisto con valuta ignota: **UNKNOWN** |
| **I020** | Esiste Purchase con valuta esplicitamente nota e diversa da EUR, in 90 giorni | Zero acquisti: **NO MATCH**. EUR+USD: **MATCH**. Solo valuta assente e osservazione completa: **NO MATCH** alla richiesta esplicita di valuta nota; non prova assenza di acquisti in altre valute reali |
| **I021** | Tutti i Purchase in 90 giorni sono EUR, con almeno un Purchase | Zero acquisti: **NO MATCH**. Solo EUR: **MATCH**. EUR+USD: **NO MATCH**. EUR+valuta ignota: **UNKNOWN** |

La variante generica senza guardia “Purchase.currency != EUR” su una occurrence con valuta mancante rimane UNKNOWN. Il requisito esplicito “noto” di I020 non viene copiato invisibilmente nelle altre condizioni.

### 28.3 Selection order

| Intento | Selezione e verifica | Fixture distintiva |
|---|---|---|
| **I025** | Primo Purchase storico; poi data in 30 giorni e EUR | USD 60 giorni fa, EUR 10 giorni fa: **NO MATCH** |
| **I026** | Primo Purchase EUR storico; poi data in 30 giorni | Stessa fixture: **MATCH** |
| **I027** | Primo Purchase entro la base degli ultimi 30 giorni; poi EUR | EUR 100 giorni fa ed EUR 10 giorni fa: **MATCH**; I025 sulla stessa fixture è **NO MATCH** |
| **I028** | Ultimo Purchase senza prefiltri; poi data in 90 giorni, EUR, >€100 | EUR €150 seguito da EUR €40: **NO MATCH**. EUR €150 seguito da USD: **NO MATCH** |
| **I029** | Ultimo Purchase dopo prefiltraggio EUR; poi data in 90 giorni e >€100 | EUR €150 seguito da USD: **MATCH**. EUR €150 seguito da EUR €40: **NO MATCH** |
| **I030** | Esistenza Purchase EUR >€100 in 90 giorni | EUR €150 seguito da EUR €40: **MATCH**; non importa che il secondo sia l'ultimo |

Con storia precedente non disponibile, “primo di sempre” non è trasformato in “primo disponibile”. Per contro, un ultimo acquisto identificabile in una coda adeguatamente osservata può essere determinabile anche se il prefisso antico non è conservato.

### 28.4 Measures

| Intento | Costruzione richiesta | Esito atteso |
|---|---|---|
| **I038** | Somma del totale pagato di tutti gli acquisti confrontabili in EUR in 90 giorni >€1.000 | Undici ordini da €99: totale €1.089, **MATCH**; I030 è **NO MATCH** |
| **I039** | Somma acquisti del periodo meno somma rimborsi del periodo >€1.000 | €1.200 acquisti e €300 rimborsi: €900, **NO MATCH**, anche se i rimborsi sono di ordini anteriori al periodo |
| **I045** | Media per ordine in 90 giorni >€100, con almeno due ordini | Undici ordini da €99: **NO MATCH**. Un solo ordine da €500: **NO MATCH** per il minimo. €120 e €100: media €110, **MATCH** |
| **I047** | Count Purchase > count operazioni Refund in 90 giorni | Due acquisti e tre rimborsi parziali, anche riferiti a un solo ordine: **NO MATCH**; il totale monetario non sostituisce i conteggi |

Un acquisto con importo ignoto non viene escluso dalla media mantenendo il nome “media degli acquisti”. Una somma monetaria non viene calcolata sommando EUR e USD come numeri privi di unità.

### 28.5 Distribution

| Intento | Costruzione richiesta | Esito atteso |
|---|---|---|
| **I040** | Count distinct feature in 30 giorni ≥5 | Cinque usi della stessa feature: **NO MATCH**. Cinque feature differenti: **MATCH** |
| **I041** | Esiste product ID con almeno tre order ID distinti in 90 giorni | Tre unità dello stesso prodotto in un ordine: **NO MATCH**. Stesso prodotto in tre ordini: **MATCH** |
| **I054** | Share delle fruizioni Scienza >60%, almeno cinque fruizioni, 30 giorni | Scienza 7 su 10: **MATCH**. Scienza 6 su 10: **NO MATCH**, perché la soglia è stretta |
| **I056** | Scienza unico valore più frequente in 30 giorni, almeno una fruizione | Quote 45%/35%/20%: **MATCH**, mentre I054 è **NO MATCH**. Scienza e Arte a pari merito: **NO MATCH** |

Una sola fruizione Scienza soddisfa I056: non si aggiunge il requisito di almeno due osservazioni. Per le quote le fruizioni ripetute restano nel denominatore.

### 28.6 Sequence e correlation

| Intento | Costruzione richiesta | Fixture distintiva |
|---|---|---|
| **I064** | View in 30 giorni AND Purchase in 30 giorni | Purchase prima della View: **MATCH** |
| **I065** | Esiste View ≺ Purchase, entrambi in 30 giorni | Purchase prima della sola View: **NO MATCH**. View prodotto A poi Purchase prodotto B: **MATCH** |
| **I066** | Come I065 con stesso product ID | View A poi Purchase B: **NO MATCH**. View A poi Purchase A: **MATCH** |
| **I067** | Esiste click A ≺ Demo, ritardo ≤48 ore, entrambi in 14 giorni | Click 4 giorni fa, Demo 3 giorni fa, nuovo click 2 giorni fa senza seguito: **MATCH** |
| **I068** | Seleziona l'ultimo click A, poi Demo successivo entro 48 ore; entrambi in 14 giorni | Stessa fixture: **NO MATCH**. Non si ripiega sul click di 4 giorni fa |

Per il caso aggiuntivo “esiste un prodotto visualizzato ma mai acquistato”, un acquisto dello stesso prodotto precedente alla view esclude il prodotto. Una condition “non acquistato dopo la view” non può essere usata come sostituto.

### 28.7 Deadline

| Intento | Costruzione richiesta | Fixture distintiva |
|---|---|---|
| **I072** | Opportunità messaggio Demo, prima consegna fra 30 e 7 giorni fa; nessun click dello stesso messaggio entro sette giorni dopo la consegna | Consegna 10 giorni fa e click all'ottavo giorno: **MATCH** |
| **I073** | Stessa base; nessun click dello stesso messaggio dopo la consegna fino a T | Stessa fixture: **NO MATCH** |

Regressioni obbligatorie: click esattamente alla deadline, osservato con T successivo, confuta I072; nessun click con T esattamente alla deadline lascia l'esito negativo UNKNOWN; click avvenuto al sesto giorno ma ricevuto tardi rettifica una precedente valutazione; click realmente all'ottavo giorno non rettifica il mancato rispetto dei sette giorni.

### 28.8 Correlated object state

| Intento | Costruzione richiesta | Fixture distintiva |
|---|---|---|
| **I074** | Esiste cart non completato, il cui ultimo aggiornamento è fra 30 giorni e 24 ore fa | Un aggiornamento 5 giorni fa seguito da uno 12 ore fa: quel cart **non soddisfa** la condition. Un acquisto di un altro cart non completa quello candidato |
| **I075** | Esiste ticket aperto nello stato attuale autorevole | Un ticket aperto e un altro chiuso: **MATCH**. Una riapertura deve essere visibile nello stato corrente |
| **I076** | Esiste ordine comprato in 90 giorni con zero rimborsi della stessa identità fino a T | Due ordini recenti, uno rimborsato e uno no: **MATCH** |
| **I077** | Tutti gli ordini comprati in 90 giorni hanno zero rimborsi fino a T, con almeno un ordine | Stessa fixture: **NO MATCH**. Nessun ordine recente: **NO MATCH** |

Un rimborso relativo solo a un ordine anteriore alla base non esclude I077. Lo stesso rimborso può però ridurre la spesa netta di I039 se avvenuto nel suo periodo.

### 28.9 Temporal patterns

| Intento | Costruzione richiesta | Fixture distintiva |
|---|---|---|
| **I080** | Almeno quattro giorni distinti nelle ultime quattro settimane complete | Quattro giorni tutti nella stessa settimana: **MATCH** |
| **I081** | Almeno un utilizzo in ciascuna delle ultime quattro settimane complete | Stessa fixture, altre tre settimane vuote: **NO MATCH** |
| **I083** | Almeno cinque avvii in una finestra continua di 24 ore, interamente negli ultimi sette giorni completi | Tre avvii alle 23 e due alle 01, all'interno del periodo: **MATCH** |
| **I084** | Almeno cinque avvii in uno stesso giorno di calendario, nello stesso periodo esterno | Stessa fixture senza altri avvii: **NO MATCH** |
| **I087** | Recenti 14 giorni ≤50% dei 14 precedenti, con precedenti ≥10 | Precedenti 10, recenti 5: **MATCH**. Precedenti 9, recenti 0: **NO MATCH** per minimo insufficiente |
| **I090** | Ritorno negli ultimi 7 giorni, con precedente uso e pausa osservata ≥30 giorni | Primo utilizzo assoluto recente: **NO MATCH**. Uso 40 giorni fa e ritorno 5 giorni fa, senza usi intermedi: **MATCH** |
| **I091** | Zero usi nell'ultima settimana completa, almeno un uso in ciascuna delle quattro precedenti | Inattività recente con una delle quattro settimane precedenti vuota: **NO MATCH**; non basta una generica storia di attività |

Se una pausa contiene un periodo non osservabile, non viene chiamata “pausa dimostrata”. Se il ritorno di I090 è certo, successive attività ordinarie non creano altri ritorni dalla stessa pausa.

### 28.10 Personal context

| Intento | Costruzione o stato | Fixture distintiva e risultato |
|---|---|---|
| **I095** | Consumo nel ciclo personale corrente / quota attuale >80% | Consumo 85, quota corrente 100: **MATCH**. Stesso consumo con quota corrente 200: **NO MATCH**. Consumo in cicli precedenti non entra nel numeratore |
| **I096** | Ultimo upgrade fra 90 e 14 giorni fa; confronto locale dopo>2×prima, minimo 10 prima | Prima 10, dopo 21: **MATCH**; dopo 20: **NO MATCH**. Un upgrade più recente fuori dalla base di recency non viene ignorato per scegliere quello precedente |
| **I097** | **DEFERRED nella baseline**; nessun costruttore o produttore assegnazioni incluso | Assegnati M1 e M2, completati M1 e un materiale estraneo: il requisito originale è **falso**, anche se i conteggi coincidono. La baseline non promette di produrne l'esito da assegnazioni/completamenti raw |
| **I098** | **DEFERRED per ricostruzione da cronologia**; filtrabile quando lo stato storico corretto è già acquisito sull'invio | Invio autorizzato allora, revoca successiva: requisito I098 **falso**; non si sostituisce con I100. La sola autorizzazione corrente è insufficiente |
| **I100** | Scope Invii con canale associato a diniego/revoca corrente dello stesso canale | Nella fixture precedente: **MATCH**. Revoca solo SMS e invio solo email ancora autorizzata: **NO MATCH**. Stato corrente ignoto pertinente: **UNKNOWN** |

La coppia I098/I100 conserva quindi entrambi i significati senza fingere disponibilità della capacità storica rinviata. La variante di I098 con snapshot storico certificato non richiede né abilita un lookup temporale generale.

### 28.11 Complete segments richiesti

#### S002 — Carrelli con interesse ripetuto: forma supportata con contratto Cart

La costruzione è una congiunzione di lingua italiana, contattabilità email e un blocco K sul carrello. Nel blocco: carrello non completato; vero ultimo aggiornamento fra 7 giorni e 24 ore fa; esiste un prodotto nel suo contenuto corrente che il profilo ha visualizzato almeno due volte negli ultimi 7 giorni.

Tutti i requisiti del cart sono sullo stesso cart; le due view sono dello stesso prodotto membro. Non è aggiunto un ordine temporale fra view e aggiornamento del cart, perché il segmento non lo richiede.

**Fixture positiva:** cart C con prodotto P, ultimo aggiornamento 3 giorni fa, non completato; due view P in 7 giorni; proprietà di profilo soddisfatte. Risultato MATCH.

**Fixture negativa:** cart C qualificante contiene soltanto P; le due view sono di Q, eventualmente contenuto in un altro cart non qualificante. Risultato NO MATCH. “Esiste cart abbandonato AND esiste prodotto rivisto” non passa questo test. Se il contenuto corrente del cart non è determinabile, non si usa una vecchia lista di prodotti per produrre MATCH.

#### S004 — Trial iniziati ma non decollati: forma supportata con contratto Trial

Un blocco K seleziona il trial corrente designato, ancora attivo e iniziato fra 14 e 10 giorni fa. Nel medesimo contesto: Import e Dashboard nei suoi primi 7 giorni; zero Share durante quel trial; zero Upgrade dall'inizio del trial fino a T. Una condizione separata richiede zero utilizzi del prodotto negli ultimi 7 giorni.

Import, Dashboard e Share devono essere attribuibili a quel trial; l'assenza di upgrade usa il periodo dall'inizio, senza una correlazione al trial aggiunta tacitamente. Il corpo impiega quattro scope subordinati: Import, Dashboard, Share e Upgrade.

**Fixture positiva:** trial corrente iniziato 12 giorni fa; Import all'inizio+1 giorno e Dashboard all'inizio+2; nessuno Share in quel trial; nessun upgrade dal suo inizio; nessun uso negli ultimi 7 giorni. Risultato MATCH.

Uno Share relativo solo a un trial precedente non esclude il segmento. Uno Share del trial corrente lo esclude. Un trial storico pertinente e un trial corrente diverso non possono essere combinati per costruire artificialmente una prova. Se manca una designazione univoca del trial corrente o un'associazione attendibile delle attività, la parte pertinente è UNKNOWN o la capacità è strutturalmente non attivabile, secondo §19.

#### S011 — Spettatori appena rientrati: forma supportata con stato corrente certificato

Il filtro richiede abbonamento pagante attivo, contattabilità email, almeno 120 minuti effettivamente guardati negli ultimi 7 giorni e una esperienza “ultimo ritorno dopo pausa di almeno 30 giorni”. Il ritorno deve avere uso antecedente, essere negli ultimi 7 giorni e, nello stesso blocco, non avere una consegna della campagna Bentornato dopo di esso fino a T.

La soglia dei 120 minuti riguarda gli ultimi 7 giorni, non un periodo ristretto arbitrariamente all'ultimo ritorno. L'evento di campagna deve rappresentare ricezione/consegna secondo il repertorio richiesto, non un semplice tentativo di invio.

**Fixture positiva:** uso 40 giorni fa, nessun uso fino a un ritorno 5 giorni fa, poi almeno 120 minuti nella settimana recente; una vecchia campagna Bentornato 20 giorni fa ma nessuna dopo il ritorno. Con gli altri requisiti soddisfatti, MATCH.

Una consegna Bentornato 3 giorni fa rende NO MATCH. Un primo utilizzo 5 giorni fa senza uso anteriore non è un ritorno. Una data piatta “ultima visione” non sostituisce la data dell'ultimo ritorno qualificante.

#### S012 — Serie lasciata incompiuta: forma condizionale a contratti e producer certificati

Il filtro richiede subscription pagante attiva, lingua italiana e un percorso sulle tappe di copertura. Il producer certificato fornisce, per ciascuno degli episodi 1 e 2 della serie A, il **primo raggiungimento del 90% cumulato a partire dall'inizio degli ultimi 30 giorni**; il filtro consuma queste milestone. Le due tappe devono avvenire in quell'ordine nel periodo. La seconda deve risalire ad almeno 7 giorni fa. Nello stesso percorso si verifica il fatto già certificato sulla seconda milestone che l'episodio 3 fosse disponibile allora; si verifica inoltre zero avvii dell'episodio 3 successivi a quella milestone fino a T. Il percorso non consulta o ricostruisce lo stato storico del contenuto.

**Fixture positiva:** prima tappa episodio 1 venti giorni fa, prima tappa episodio 2 dieci giorni fa, episodio 3 disponibile già allora, nessun suo avvio da quella seconda tappa. Con stato e lingua soddisfatti, MATCH.

**Regressioni:** se la prima tappa dell'episodio 2 nel periodo precede quella dell'episodio 1, non si seleziona una successiva rivisione dell'episodio 2 per recuperare il percorso. Se l'episodio 3 è disponibile oggi ma non lo era allora, NO MATCH; se la disponibilità storica è ignota, UNKNOWN. Un suo avvio antecedente alla seconda tappa non è escluso dal segmento; un avvio successivo sì.

Un semplice evento lifetime “Completed episode 2” non passa il contratto, a meno che l'equivalenza con la tappa richiesta nel periodo sia certificata. Senza il producer necessario questa costruzione è non attivabile; con producer presente ma milestone o disponibilità insufficientemente determinate segue UNKNOWN dove l'incertezza è decisiva. Non si attribuisce questo caso al solo core language.

#### S013 — Rinnovi da seguire con priorità: forma supportata con contratto Ticket

Il filtro è: piano corrente in Pro/Enterprise; rinnovo nei prossimi 14 giorni; ultima risposta NPS con voto ≤6; utilizzi recenti 14 giorni ridotti di almeno 50% rispetto ai 14 precedenti, con almeno 10 precedenti; esiste almeno un ticket attualmente aperto.

Il requisito di rinnovo è una proprietà temporale corrente, non una previsione del motore. Il calo è personale, non relativo alla popolazione. Stato del ticket e ultima risposta NPS sono fatti indipendenti; non devono appartenere a una stessa esperienza.

**Fixture positiva:** Pro, rinnovo fra 7 giorni, ultimo NPS 5, utilizzi precedenti 10 e recenti 5, un ticket aperto. Risultato MATCH.

Un NPS precedente 5 seguito da ultimo NPS 9 rende NO MATCH. Precedenti 9 e recenti 0 non soddisfano il minimo richiesto. Un ticket chiuso non annulla l'esistenza di un altro ticket aperto.

#### S014 — Ticket chiusi, esperienza non ripresa: forma supportata con stato corrente certificato

Il filtro richiede stato pagante attivo, zero ticket aperti oggi e un blocco E sulla **ultima chiusura di ticket del profilo**, selezionata prima dei controlli. Nel blocco: data fra 14 e 7 giorni fa; almeno un utilizzo nei 30 giorni precedenti quella chiusura; zero utilizzi successivi fino a T.

Il repertorio di utilizzi è quello del prodotto richiesto; non si aggiunge una chiave ticket agli utilizzi. È l'ancora temporale a dover rimanere la stessa nelle due finestre.

**Fixture positiva:** ultima chiusura 10 giorni fa; uso 11 giorni fa e nessuno dopo la chiusura; nessun ticket aperto; stato pagante attivo. Risultato MATCH.

Una nuova chiusura 3 giorni fa rende NO MATCH perché cambia l'ultima chiusura, anche se quella di 10 giorni fa avrebbe soddisfatto gli altri requisiti. Un uso 5 giorni fa rende NO MATCH. L'assenza di ticket aperti non può essere inferita dall'esistenza di una sola chiusura recente.

### 28.12 Regressioni supplementari della baseline

| Caso | Esito semantico da preservare |
|---|---|
| I031 | Ultimo NPS6, precedente 9: calo di 3, MATCH; non confronto fra medie |
| I033 / I034 | Terzo storico e terzo nel periodo possono avere date differenti; un quarto evento non annulla la terza tappa |
| I035 | Un Export precedente all'iscrizione o fuori dalla finestra non viene ignorato per selezionare un primo Export qualificante nei 7 giorni |
| I037 | Primo contatto 24 ore dopo la creazione e altro contatto 72 ore dopo: NO MATCH al “primo oltre 48 ore”; assenza totale di contatti non è un contatto tardivo |
| I048 | Dieci messaggi, tre cliccati entro 7 giorni, anche con molti click sugli stessi tre:30%, MATCH; denominator 10, non numero di click o sole opportunità risolte |
| I050 | Ordine pagato 100, rimborsi 30+70 dello stesso ordine: MATCH; rimborso 30 e altri 70 su un altro ordine non bastano |
| I051, con producer certificato | Il producer attesta copertura unica 10% nonostante dieci rivisioni della stessa porzione: con evidenza completa, NO MATCH alla soglia 90%. Il filtro non ricostruisce il calcolo |
| I069 | Creazione→Import→Share sullo stesso project, entro 7 giorni totali; importare su un altro project non completa il percorso |
| I070 | Errore su altro project o fuori dagli estremi non esclude; errore del project fra creazione e condivisione esclude quella coppia |
| I071 | Un'altra azione utente del repertorio prima del contatto esclude l'adiacenza; un evento automatico escluso dal repertorio non la esclude |
| I078 | Tre ticket maturi, due risposti in 48 ore e uno in 49: NO MATCH al “tutti entro 48 ore” |
| I079 | Close→Reopen→Close→Reopen su un solo ticket vale due cicli; distribuire i cicli su ticket diversi non basta |
| I082 | Dodici giorni attivi che lasciano una delle quattro settimane con meno di 3 giorni non bastano |
| I085 / I086 | Una serie già interrotta può soddisfare lo streak; un gap iniziale di 8 giorni confuta il limite massimo 7 anche se tutti i gap interni sono brevi |
| I088 | Settimane 12,8,9: NO MATCH, anche se l'ultima è inferiore alla prima |
| I092 | Una pausa con un solo ritorno conta una volta, anche se molti utilizzi storici possono precederla |
| I093 | Ultimi intervalli 10 e 20 giorni: MATCH al “più recente almeno doppio”; un intervallo zero non soddisfa il prerequisito di positività |
| I099 | Tre campagne distinte ordinate per prima consegna; click su due delle tre vale MATCH; click solo su campagne precedenti non basta |
| Identità multivalore nel percorso | Tre passi con chiavi {A,B}, {B,C}, {A,C} non hanno una chiave comune all'intera tupla: NO MATCH alla correlazione unica globale |
| Deduplicazione | Tre copie della stessa azione non soddisfano “almeno tre volte” |
| Storia incompleta | Tre acquisti certi più altri possibili: MATCH ad “almeno tre”, UNKNOWN a “esattamente tre” |
| Booleani ternari | NO MATCH AND UNKNOWN = NO MATCH; MATCH OR UNKNOWN = MATCH; nessun coercimento globale degli UNKNOWN a falso |

Le costruzioni illustrate conservano gli esiti attesi sotto i rispettivi prerequisiti. Le condizioni di stato corrente e le evidenze derivate mantengono le dipendenze di §26. I097/S015 e la ricostruzione di I098 restano deferred; le fixture che mostrano perché una scorciatoia è scorretta non costituiscono un PASS di supporto.

### 28.13 Scoped correlation: O03 e regressioni di ambito

| Fixture / configurazione | Risultato atteso |
|---|---|
| Primo ordine P1; Refund di P1; successivo nuovo Purchase P2; blocco order ID sui soli passi 1–2 | MATCH. Non è richiesto che P2 condivida l’order ID di P1; il blocco non impone neppure una disuguaglianza al passo esterno |
| Primo ordine P1; rimborso soltanto di P2; nuovo acquisto P3 | NO MATCH, con storia e chiavi complete. Non si cambia il primo ordine scelto |
| P1 e Refund P1 seguiti da nuovo acquisto privo di order ID, ma con identity di occurrence e proprietà necessarie note | La sola assenza di order ID nel passo 3 non impedisce MATCH: il passo è esterno al blocco |
| Tre passi nel blocco con chiavi {A,B}, {B,C}, {A,C} | NO MATCH alla chiave comune, pur esistendo intersezioni pairwise |
| Blocco sui passi 1–2 con {A,B} e {B,C}; passo 3 esterno con {A,C} | Il valore B soddisfa la correlazione del blocco; il passo 3 non interviene. L'esito complessivo dipende dagli altri requisiti |
| Blocchi sui passi 1–2 e 3–4, oppure product ID sul primo arco e order ID sul secondo | Invalid configuration, non UNKNOWN e non NO MATCH |
| Esito dal passo 3 esterno al blocco 1–2 che tenta di leggere la chiave locale del blocco | Invalid configuration; nessuna propagazione implicita |

### 28.14 Named evidence scopes: O17

Si valutano inventario e stato corrente certificati, apertura iniziale >48 ore fa e zero Human Response dalla stessa apertura fino a T.

| Fixture | Risultato atteso |
|---|---|
| Ticket A aperto da 72 ore senza risposte; ticket B con risposta umana | MATCH: la risposta di B non riguarda A |
| Unico ticket A aperto da 72 ore con risposta umana dalla sua apertura | NO MATCH |
| Unico ticket A aperto inizialmente da 72 ore, riaperto 2 ore fa, nessuna risposta nel lifecycle | MATCH: l'età non è quella della riapertura |
| Unico ticket aperto esattamente da 48 ore | NO MATCH alla soglia stretta “più di 48 ore” |
| Ticket A aperto da 72 ore; nessuna risposta osservata, ma Human Response non completamente osservabile | UNKNOWN, salvo altro ticket certamente qualificante |
| Una condizione trova A aperto e una condizione sorella prova assenza di risposte su B | Non costituisce una prova di O17; una congiunzione non condivide il ticket |
| Contratto Ticket privo dello scope necessario e nessuna forma conforme disponibile | Capability non attivabile; non si usa un join scelto dall'utente |

### 28.15 Qualified-period cardinality: O15

Per “almeno 2 giorni attivi in almeno 3 delle ultime 4 settimane complete”, le cifre indicano giorni distinti di ascolto per settimana, non numero di ascolti.

| Quattro settimane / evidenza | Risultato atteso |
|---|---|
| 2, 2, 2, 0 giorni; osservazione completa | MATCH; “almeno 2 giorni ogni settimana” è NO MATCH |
| 6, 0, 0, 0 giorni | NO MATCH; sei giorni complessivi non compensano le tre settimane |
| 2, 2, 1, 0 giorni | NO MATCH: solo due settimane qualificano |
| Tre settimane certamente qualificanti, quarta non osservabile | MATCH al minimo 3; la quarta resta UNKNOWN, non viene cancellata |
| Due settimane qualificanti, una non qualificante, una non osservabile | UNKNOWN: il numero qualificante è fra 2 e 3 |
| Una settimana qualificante, due non qualificanti, una non osservabile | NO MATCH: il massimo possibile è 2 |
| Intervallo “tra 2 e 3 settimane”: due qualificanti, una non qualificante, una incerta | MATCH: tutte le cardinalità possibili sono nell'intervallo |
| Elimina la settimana vuota e sostituiscila con una più antica | Configurazione non conforme; cambia la serie richiesta |

### 28.16 Operandi atomici, terminal arithmetic e part/whole

| Caso | Risultato atteso |
|---|---|
| Due Purchase e tre operazioni Refund nel periodo | NO MATCH a count Purchase > count Refund; il numero di ordini rimborsati non sostituisce RefundCount |
| Dieci ordini, due operazioni di rimborso sul medesimo ordine | Ratio operazioni Refund/ordini =20%; share ordini con almeno un rimborso =10%. Sono domande diverse |
| Media durata recente 12, precedente 10, almeno cinque job per periodo | MATCH al confronto delle medie; eventuale requisito sugli errori resta una condizione distinta, come O07 |
| Sum acquisti €1.200 e credit limit corrente €1.000 | MATCH al confronto diretto, con unità e periodo pertinenti |
| Ratio con denominatore certamente zero | NO MATCH per guardia di positività non soddisfatta; niente infinito o zero artificiale |
| Ratio con denominatore ignoto e senza altra prova decisiva | UNKNOWN |
| Ratio di due ratio, difference divisa per C, addizione di min e max | Invalid configuration; nessuna recipe può renderle operandi atomici |
| Scalare esterno certificato valido a T | Filtrabile nello slot ammesso; non esporta le occurrence originarie e non rende nativo il suo algoritmo |

### 28.17 Certified derived evidence e separazione delle responsabilità

| Caso | Risultato atteso |
|---|---|
| I051 con producer presente, copertura unica certificata ≥90% nella finestra corretta | Il normale confronto può produrre MATCH |
| Solo VideoStarted o massima percentuale raw del player, senza contratto equivalente | Non costituisce supporto a I051; la capability dipendente dal producer è non attivabile |
| Producer presente, soglia attestata ma primo timestamp pertinente non certificabile | Il test di soglia può essere determinato; una selezione/percorso che dipende da quel timestamp può essere UNKNOWN |
| S012: episodio 3 disponibile oggi, certificato non disponibile alla seconda milestone | NO MATCH |
| S012: disponibilità alla seconda milestone ignota, altri requisiti soddisfatti | UNKNOWN, non lookup storico automatico |
| Snapshot di autorizzazione dell'invio certificato positivo allora e diniego corrente | Non soddisfa il requisito negativo storico I098; può soddisfare I100 |
| Scadenza di validità di uno scalare derivato dichiarato corrente prima di T, senza nuova evidenza valida | Non si conserva un MATCH corrente sulla sola vecchia evidenza |

### 28.18 Contesti, ternarietà e perimetro

| Caso | Esito o vincolo atteso |
|---|---|
| MATCH AND UNKNOWN | UNKNOWN |
| NO MATCH AND UNKNOWN | NO MATCH |
| MATCH OR UNKNOWN | MATCH |
| NO MATCH OR UNKNOWN | UNKNOWN |
| Un profilo prima MATCH diventa UNKNOWN alla valutazione corrente | Escluso dalla membership corrente; nessun last known match |
| Una recipe espansa sposta una foglia fuori dal proprietario dell'ancora | Invalid configuration |
| Si pubblica la data di un evento come scalare e una sorella tenta di leggere l'order ID di quell'evento | Riferimento vietato; lo scalare non recupera il testimone |
| O08 con soli eventi individuali e relazioni account | Fuori dal linguaggio; nessun join fra profili |
| O16 con ranking già pubblicato, platea/parità e validità dichiarate | Il dato esterno è consumabile; il ranking dinamico non viene calcolato dal filtro |
| Lettura tecnica fallita | Execution failure; non un nuovo truth value né UNKNOWN semantico inventato |

Questi test non aggiungono intenti al benchmark né ne cambiano le classificazioni. Verificano i confini e le distinzioni già definiti nel corpo normativo.

---

### 28.19 Regressioni di raccordo RF (proposta 1.1)

Le verifiche seguenti riguardano le condizioni condivise e non aggiungono nuovi intenti al benchmark. Quando è indicato un risultato SQL, si intende la valutazione dell’espressione prima dell'eventuale `WHERE` esterno.

| Caso | Risultato atteso |
|---|---|
| `amount is not 100`, proprietà ordinaria Profiles assente | SQL `NULL`; Go `Applies=false` |
| `properties.amount is not 100`, percorso JSON assente | SQL `NULL`; Go `false` |
| `properties.amount is not 100`, JSON `null` presente | SQL TRUE; Go `true` |
| `properties.amount is 100`, JSON String `"100"` presente | SQL FALSE; Go `false` |
| `properties.amount is greater than 100`, JSON String `"200"` presente | SQL `NULL`; Go `false` |
| `properties.amount does not exist`, percorso JSON assente | SQL TRUE; Go `true` |
| `properties.amount is empty`, percorso JSON assente | SQL TRUE; Go `true` |
| `properties.amount is one of [null]`, JSON `null` presente | SQL TRUE; Go `true` |
| `properties.amount is one of [null]`, percorso JSON assente | SQL `NULL`; Go `false` |
| `(TRUE AND NULL)` / `(TRUE OR NULL)` in SQL | NULL / TRUE |
| Source/Database/User della baseline con `core.Filter` | Rifiuto di validazione, non attivazione Unified |
| Regola E/K/P/T in una pipeline record-only | Rifiuto esplicito server, non applicazione parziale |
| Quattro gruppi principali con alternanza AND/OR entro 100 discendenti | Forma ammessa; non ridotta obbligatoriamente a tre livelli |
| Gruppo principale esplicitamente vuoto | Invalid configuration |

Il record filtering Go continua a restituire soltanto `bool`, mentre la diagnostica ternaria per Unified è definita sul filtro complessivo. I renderer PostgreSQL/Snowflake devono preservare il contratto SQL; Snowflake remoto resta una verifica di integrazione non attestata per la baseline indicata.

## 29. Evolution rules

### 29.1 Estensioni compatibili

MAY essere introdotti nuovi operatori scalari tipizzati, riduzioni atomiche nominate, preset finiti, recognizer temporali chiusi, ruoli di identità, contratti mirati o producer certificati. Ogni estensione MUST avere una definizione esplicita di input, unità/grana, scope, slot, vuoto, incertezza, ordine e tempo pertinenti, oltre ai casi che la distinguono dalle forme esistenti.

L'estensione MUST NOT cambiare la semantica di condizioni già salvate, trasformare un terminale in un operando, unire i corpi autorizzati in un body universale, esportare testimoni o allargare implicitamente un blocco di correlazione. Un nuovo pattern MUST essere una forma nominata, non accesso a un algoritmo temporale generale.

Aumentare un limite numerico può essere compatibile quando preserva ruoli, confini e significato; non autorizza nuove combinazioni. Una formula attualmente vietata non diventa ammessa perché una futura UI ne faciliti l'inserimento: richiederebbe una nuova forma chiusa esplicitamente approvata e versionata.

### 29.2 Producer deferred compatibili, non inclusi

**I097 / S015:** un futuro producer mirato di completamento degli assegnati del percorso corrente MUST preservare insieme corrente, almeno un assegnato, identity materiale–percorso e completezza. Il semplice confronto fra conteggi non è sostitutivo.

**I098:** un futuro producer specifico può associare agli invii lo stato di autorizzazione documentato allora da una cronologia sufficiente. Non introduce un lookup “as of” universale.

**I094:** un futuro valore governato deve preservare il ritardo rispetto alla mediana degli intervalli degli ultimi cinque acquisti, cinque acquisti, quattro intervalli e mediana positiva come riportato dalla Fase 3. Non autorizza un editor statistico inline.

Un futuro dominio account/related-entity o risultato derivato autorevole può dare accesso a fatti collettivi con un contratto esplicito; non è incluso qui e non può essere anticipato mediante correlazioni fra profili.

### 29.3 Stabilità delle definizioni

Recipe, mapping, domain contract e producer MUST conservare una definizione riconoscibile per i filtri salvati. L'adozione di un significato nuovo richiede una scelta esplicita e differenze leggibili. MUST NOT essere cambiati silenziosamente grana, repertorio, calendario, periodo, selezione, correlazione, guardie o regole di evidenza.

La scelta di una fonte concreta è una configurazione e certificazione di un contratto già definito, non il luogo in cui inventarne il significato. L'esecuzione MAY ottimizzare o precomputare, ma MUST preservare risultato e dipendenze; non può troncare la storia, usare selezione greedy, scartare unknown o omettere periodi vuoti per ragioni di costo.

### 29.4 Conformità della traduzione tecnica

Una realizzazione conforme MUST consentire al product designer di scegliere organizzazione e presentazione senza inventare semantiche; all'API designer di rappresentare solo le forme e gli slot ammessi; a implementazioni indipendenti di determinare gli stessi MATCH/NO MATCH/UNKNOWN sugli stessi dati, contratti, parametri e T; al business user di esprimere gli intenti supportati senza join, group by, variabili, expression tree o query planner.

Con dati completi possono comunque esserci UNKNOWN se un campo è realmente ignoto, l'ordine pertinente non è certificato o una deadline è futura: “completi” non crea un fatto inesistente. La precisione numerica/temporale e le condizioni limite sono quelle normative, non scelte discrezionali del backend.

La UI dettagliata, la rappresentazione tecnica e il comportamento esecutivo restano scelte di realizzazione **entro** questi vincoli. Le questioni di formato, persistenza e raccordo tecnico elencate in chiusura devono essere risolte esplicitamente prima della Fase 4B, senza introdurre cambiamenti impliciti al significato delle condizioni.

---

## 30. Decisioni normative riconciliate (proposta 1.1)

Questa sezione è un indice delle decisioni consolidate e degli adattamenti **proposti**. Le definizioni operative sono quelle delle sezioni indicate, non alternative aggiuntive. La versione 1.0 resta il riferimento congelato fino all’approvazione della proposta 1.1.

| Aspetto / decisione proposta | Sezioni |
|---|---|
| Unico `core.Filter`; record filtering incluso in Unified; ammissibilità contestuale | §§1.3, 2, 26.1.1 |
| Catalogo condiviso dei 26 operatori e semantica aggiornata dei missing/JSON | §§8.1–8.2, 19.1–19.2 |
| Quattro gruppi principali e 100 discendenti, senza riscrittura automatica del documento pubblico | §§7.5, 20 |
| Cinque primitive e separazione user/semantic/execution model | §§2–6 |
| Località lessicale, corpi chiusi, ruoli, slot, limiti e divieto di export | §7 |
| Proprietà correnti, same-occurrence, grana e periodi | §§8–9 |
| Operandi atomici, arithmetic terminale, share part/whole, distribuzioni chiuse | §10 |
| Selezione prima della verifica, assenza di fallback, first/last/nth e preset | §11 |
| Forme di candidati, quantificatori, non-vuoto e opportunità | §12 |
| Tuple finite, ordine, durata, negative outcomes e scoped correlation block | §13 |
| Catalogo temporale chiuso e qualified-period cardinality | §14 |
| Stato autorevole, named evidence scopes e slot specifici del contratto | §15 |
| Consumo di Certified Derived Evidence, distinto dagli algoritmi di dominio | §16 |
| T, calendario, ancore, estremi e distinzione corrente/storico | §§17–18 |
| UNKNOWN, completezza, motivi diagnostici, invalidità, failure e membership corrente | §19 |
| AND/OR ternari e assenza di NOT generico | §20 |
| Recipe, catalogo e stabilità del significato | §§21–22 |
| Inline/derived e confini filtering/analytics/automation | §§23–24 |
| Discoverability senza perdita di significato | §25 |
| Coverage separata, dipendenze, deferred ed esclusioni | §§26–27 |
| Esiti di accettazione e regole di evoluzione | §§28–29 |

Le decisioni elencate non attestano la disponibilità di dati, contratti o producer in un’installazione concreta. Le capacità deferred o escluse non sono TODO impliciti del linguaggio. **L’approvazione della nuova revisione è separata da questa bozza.**

---

## Consolidation changelog

*Sezione informativa. Riporta soltanto modifiche rispetto alla Fase 3 originale; non definisce semantica concorrente.*

### C1 — Scoped correlation block

La chiave unica del percorso può ora riguardare l'intero percorso oppure un solo blocco contiguo di almeno due passi. Sono definiti il mancato trasferimento della chiave ai passi esterni e gli slot negativi/finali che possono leggerla. Restano vietati blocchi multipli, chiavi per archi differenti e formule di join. Sostituiti Fase 3 §§5–6, 12.4, le righe di capability/acceptance e i riepiloghi che imponevano la chiave su tutti i passi; semantica integrata in §§7 e 13.

### C2 — Contract-defined named evidence scopes

Gli Authoritative Object Contracts possono esporre un elenco finito di scope nominati automaticamente correlati alla stessa object identity. Ticket può combinare stato corrente, apertura iniziale e Human Response senza campo di prima risposta precomputato; Order può dichiarare Refund. Restano vietati join scelti dall'utente, navigazione arbitraria, nuovi contesti liberi e una capability speciale Ticket Response. Sostituita la restrizione Fase 3 §15.2 che limitava Ticket a stato/attributi senza questi scope; aggiornati corpi, catalogo e acceptance.

La coerenza con S004 è preservata distinguendo il preesistente slot temporale Upgrade del profilo dal named evidence scope correlato Usage del trial. Non è stata aggiunta una correlazione trial ID agli Upgrade. Questa precisazione è necessaria affinché la correlazione automatica dei nuovi scope non alteri un requisito precedente; definita in §15.7.

### C3 — Qualified-period cardinality

Aggiunta la conta di periodi calendar qualificati da un recognizer chiuso, con serie finita, vuoti mantenuti, cardinalità minima/intervallo e propagazione dei periodi UNKNOWN. Il sotto-catalogo qualificante e il divieto di auto-annidamento sono espliciti. Restano vietati predicati liberi sulla time series, trasformazioni o nuovi raggruppamenti dei bucket. Estesi Fase 3 §14 e relativi cataloghi, limiti di composizione e acceptance; semantica integrata in §14.7.

### C4 — Terminal arithmetic

Separati operandi atomici e test terminali. Difference, ratio e relative change non producono valori riutilizzabili inline; le soglie sono parametri del test chiuso e share conserva part/whole. Prevalenza e quota massima non diventano nuovi operandi generici. Esplicitato che le durate native del preset degli ultimi intervalli non sono risultati di difference concatenati. Restano vietati expression tree, addizione libera, ratio di ratio e riuso aritmetico di risultati terminali. Sostituiti i passaggi Fase 3 §§5, 9 e 11 che potevano essere letti come un catalogo ricorsivo di misure; aggiornata la strategia UI.

### C5 — Certified Derived Evidence

La precedente capacità video obbligatoria è sostituita dal consumo generale e tipizzato di evidenze derivate certificate: scalari, occurrence, stati e milestone. Il video diventa un'applicazione condizionale di dominio, con milestone e fatti storici certificati come input ordinari. Restano vietati conoscenza dei segment ranges, unione delle porzioni nelle primitive, slot video, funzioni storiche verticali e recupero del testimone da uno scalare. Sostituiti Fase 3 §§1, 6, 16, 20.3–20.4, 21, 24, 25/S012 e 28 nelle parti che imponevano il producer video o gli attribuivano uno slot speciale.

### C6 — Chiusura esplicita dei contesti e responsabilità delle forme

La località già presente è resa parte di un vincolo congiunto con corpi chiusi, ruoli tipizzati, whitelist di slot e limiti strutturali. Chiarito che K unifica il modello di candidato, non l'algebra delle fonti, e che l'unione degli slot non è un body universale. Precisata l'applicazione del limite di quattro scope alla costruzione, incluso l'eventuale membro autorizzato, senza contare nuovamente una guardia sul medesimo scope. Restano vietati variabili nominabili, subquery generiche ed export tramite recipe/pubblicazioni. Rielaborati Fase 3 §§5–6; nessuna nuova primitiva introdotta.

### C7 — Coverage distinta per dipendenza

Rimosse le dichiarazioni aggregate come promessa incondizionata o copertura core; separate core language, targeted domain contracts e certified derived evidence. I051 e S012 sono condizionali ai producer pertinenti, non produttori assorbiti nel core. Restano vietati score sostitutivi opportunistici e riclassificazione degli intenti; i deferred precedenti rimangono tali. Sostituiti Fase 3 §§1, 24 e 28 e le qualifiche di supporto negli acceptance.

### C8 — Membership corrente e diagnostica

Esplicitata la transizione MATCH → UNKNOWN con rimozione dalla membership corrente e il divieto di last known match. Organizzate le ragioni di UNKNOWN senza nuovi truth values, mantenendo la distinzione da invalid configuration, capability non attivabile ed execution failure. Restano vietati fallback a risultati precedenti come verità corrente e conversione dei failure in esiti semantici. Integrata Fase 3 §17 e aggiunte regressioni in §28.18; le truth table preesistenti sono conservate.

### C9 — Consolidamento editoriale e acceptance coerenti

Rimosse alternative precedenti, verdetti concorrenti, riferimenti a un producer video obbligatorio e formule di coverage superate. Riorganizzati confronto, distribuzioni, stato storico e ancore senza cambiare i significati preesistenti; la tabella dei bordi distingue ora esplicitamente ancora-data e ancora-occurrence, preservando l'ordine affidabile a timestamp uguali già previsto. Inserite le regressioni delle cinque decisioni e dei casi O03, O15 e O17, senza nuovi intenti o una nuova matrice completa. Restano vietate semantiche definite soltanto nel changelog e dichiarazioni di review o test eseguiti non presenti negli allegati.

---

## Changelog di riconciliazione — proposta 1.1 (informativo)

- Unificato il contratto pubblico di riferimento: `core.Filter` per Record e Unified; introdotta la matrice dei contesti ammessi (§§1.3, 26.1.1).
- Conservati i 26 operatori, i loro payload e la semantica successiva a `d69e6168`, con risultato SQL nullable per i 20 operatori sui valori quando manca la proprietà (§§8.1–8.2, 19).
- Esplicitata la distinzione fra SQL `NULL`, JSON `null` presente e `false` di applicabilità Go; i predicati dello scope comportamentale preservano i casi incerti nelle quantificazioni (§§9.2, 19).
- Elevato il limite della radice a **4** gruppi e fissato il budget di **100** nodi discendenti, senza alterare i limiti interni delle nuove costruzioni (§7.5).
- Inserite regressioni di raccordo (§28) e preservate le primitive chiuse V/E/K/P/T, i contratti di evidenza e i vincoli sui testimoni.

**Rinvii ancora aperti per la Fase 4B:** formato wire e persistito delle nuove `FilterRule`, dettagli della rappresentazione tipizzata dei 26 operatori e delle reference raw/governate, uguaglianza e canonicalizzazione nei diversi livelli, snapshot/coerenza delle richieste Unified, divergenze residue fra executor e verifica Snowflake reale. Non sono decisioni introdotte implicitamente da questa revisione.

## Dichiarazione di revisione della specifica (proposta 1.1)

La proposta 1.1 **mantiene** le cinque primitive e le loro responsabilità, raccordandole al catalogo di record filtering esistente; le fonti e i contesti ammessi; località, corpi chiusi, ruoli, slot e limiti; confronto tipizzato e riduzioni atomiche; aritmetica terminale e share part/whole; selezione prima della verifica; candidati e quantificatori; percorsi con un solo scoped correlation block; catalogo temporale e qualified-period cardinality; contratti autorevoli con named evidence scopes; consumo di Certified Derived Evidence; modello temporale e distinzione corrente/storico; logica ternaria, ragioni di UNKNOWN e membership corrente; recipe/catalogo; confini inline/derived, analytics/automation e reporting delle dipendenze.

Sono compatibili soltanto estensioni esplicitamente definite e versionate di operatori tipizzati, riduzioni atomiche, preset e recognizer chiusi, limiti, mappings, domain contracts e producer certificati che rispettino §29 e non cambino i significati esistenti. L’adattamento dei 26 operatori della baseline è un **raccordo esplicito**, non una promozione di nuove capacità implicite. Le capacità deferred non sono incluse automaticamente e non autorizzano nuove forme implicite.

Restano intenzionalmente fuori dal linguaggio formule arbitrarie, join e chiavi multiple per archi, variabili cross-condition, navigazione illimitata, query ricorsive, ranking/percentile dinamico di popolazione, confronti nth-to-nth arbitrari, statistiche e time-series expressions generali, lookup storico universale, correlazioni comportamentali fra profili, NOT generico e live workflow semantics. Gli algoritmi domain-specific, inclusa la produzione dell'evidenza video, restano responsabilità di producer esterni al core.

**AST, API, UI dettagliata ed esecuzione MUST rispettare questa specifica e MUST NOT ridefinirne la semantica. Il lavoro successivo è esclusivamente la traduzione del modello semantico congelato in una specifica tecnica.**
