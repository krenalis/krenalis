# Behavioral Filtering — Canonical Semantic AST & Type System
## Fase 4A · Canonical AST riconciliato — versione 1.2 — Approvata

**Stato:** **versione tecnica 1.2 approvata esplicitamente, incluse le correzioni N1, N2 e N3**. La 1.2 è la nuova baseline tecnica del Behavioral Profile Filtering. La versione 1.1 approvata nel commit `25f639d84eb35815af5d1062c86545b65b97d720` rimane riferimento storico. Il presente testo adatta il significato delle forme esistenti al mondo chiuso degli eventi; non implementa la Fase 4B e non approva il rapporto architetturale 12.

**Base normativa:** [Specifica semantica riconciliata — versione 1.2, approvata](06-specifica-semantica-riconciliata-v1.2.md), derivata dalla 1.1 approvata il 27 ottobre 2026 e dalla successiva decisione di prodotto sul mondo chiuso. I documenti 1.0 e 1.1 restano conservati. **Tutti i rinvii normativi interni al documento 07 si riferiscono alla specifica 06 v1.2**, salvo richiamo espressamente storico. L’approvazione della 1.2 include espressamente la quantificazione composizionale N2, senza inferenza globale fra candidati K distinti e preservando le dipendenze interne obbligatorie.

**Ambito:** rappresentazione semantica canonica, tipi, ownership lessicale, riferimenti ai contratti, invarianti statiche e normalizzazione. Non sono definiti wire format, nomi pubblici di campi, JSON Schema, API, protocollo di compatibilità o piano di esecuzione. La notazione è un ADT ML-like: i nomi sono tecnici e provvisori.

**Emendamento approvato dall’utente il 13 novembre 2026:** la deroga di §14.4.1 conserva le regole di confronto del warehouse per i predicati Record ordinari su testo nativo. È un’eccezione esplicita all’invarianza rispetto all’executor, non una nuova garanzia sulle collation né una modifica dei costruttori AST.

**Decisione centrale:** il modello canonico per Behavioral è un’algebra di risultati di profilo con cinque sole primitive `V | E | K | P | T`. I predicati del record filtering sono **forme del confronto V o foglie autorizzate nei body**, non una sesta primitiva. L’input pubblico resta `core.Filter` condiviso con il filtraggio di singoli record. I valori atomici, gli scope, i quantificatori, i contratti, le guardie e i recognizer interni sono helper tecnici; nessuno restituisce una query, formula o testimone per una rule sorella.

---

## 1. Invarianti da preservare prima di scegliere i tipi

| Invariante semantico | Conseguenza per la rappresentazione |
|---|---|
| Ogni rule **Behavioral** riguarda il profilo corrente allo stesso T | Il soggetto non è un parametro selezionabile degli scope; T è ambientale e unico |
| Solo MATCH determina membership | Nessun campo `includeUnknown`, fallback al risultato precedente o strategia di null configurabile |
| V ha operandi atomici e operazioni terminali | Due famiglie di tipi disgiunte; un terminale non implementa l'interfaccia di un operando |
| I predicati di uno scope riguardano la stessa occurrence | I field reference sono tipizzati rispetto allo schema dell'occurrence di quello scope |
| Selezione prima della verifica | Base, posizione e body sono componenti distinti di E; nessuna promozione di verifiche a prefiltri |
| La fonte dei candidati precede il body | K conserva separatamente candidate base, eventuale selezione e quantificatore |
| Quota e rapporto non sono equivalenti | Share ha una sola base whole e una qualificazione part; ratio ha due operandi indipendenti |
| Identità business, identità di deduplicazione e grana sono distinte | Riferimenti e contratti distinti; nessuna stringa unica `key` |
| La località da sola non basta | Ogni owner ha una union chiusa di foglie; non esiste `Body<AnyRule>` |
| Un percorso ha zero oppure un blocco correlato | Un solo costruttore di blocco, con prefix e suffix non correlati |
| I pattern temporali sono enumerati | Nessun tipo generale per espressioni temporali o serie |
| Stato corrente e fatti storici sono differenti | Tipi di accesso differenti; nessun lookup storico implicito |
| Il contratto autorizza l'uso, non solo la lettura del dato | Riferimenti indicizzati per forma dell'evidenza, owner e consumer |
| Le limitazioni di struttura sono parte del linguaggio | Tipi limitati per profondità, arità, posizioni, numero di scope e member access |
| Mondo chiuso delle occurrence event-based | Solo eventi registrati interrogabili; vuoto non è indisponibilità e i dati esistenti possono restare incerti |
| Una recipe non completa il significato | L'AST e le definizioni semantiche referenziate bastano senza metadata UI |

I test di prevalenza, cambiamento di prevalenza e quota massima per valore appartengono al catalogo normativo di V: non devono scomparire perché non sono una delle cinque forme aritmetiche. Analogamente, l'accesso esistenziale a un membro posseduto dell'occurrence non deve essere confuso con il passaggio Cart → Product.

**Riferimenti normativi (specifica 06 v1.2):** §§3, 6–7, 9–16, 18–20, 27.

### 1.1 Questioni interpretative e informazioni non fornite

La revisione 1.1 ha risolto incompatibilità effettive fra catalogo Record e AST originario. La versione 1.2 conserva quelle decisioni e cambia il dominio event-based secondo 06 §3.1: **eventi registrati, disponibili e attribuiti nella vista coerente**, senza certificare la relazione reale. I costruttori AST restano invariati; si aggiornano semantica dei descrittori storici, requisiti di evidenza e fixture. Nessun flag di mondo aperto/chiuso viene aggiunto. Le correzioni successive alla revisione indipendente aggiungono il vincolo degli estremi finiti per MaximumGap (§13.3), la scelta composizionale per K (§9.1) e la valutazione della verità dell’intero test di prevalenza (§7.4), senza cambiare le dichiarazioni ADT. N1, N2 e N3 sono state approvate esplicitamente insieme alla versione 1.2. Due punti della riconciliazione restano validi:

- La scrittura abbreviata `[upgrade−Δ, upgrade)` non sostituisce la regola specifica di §17.3 per un'ancora-occurrence: contano precedenza effettiva e ritardo, incluse le parità temporali ordinabili. Il tipo conserva questa regola, non il solo intervallo di timestamp.
- Il catalogo del record filtering **è ora noto**: 26 operatori, rappresentazione `core.Filter` e semantica aggiornata dal commit `d69e6168`. Massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. Il raccordo è definito in §§2–4, 14 e 18. Una conversione fisica o persistenza lossless non è attestata senza progettare il formato e il protocollo della Fase 4B; non è introdotta una modalità semantica legacy.

Nei mapping esemplificativi, i parametri omessi dalle frasi sono dichiarati esplicitamente come assunzioni illustrative; non sono default business introdotti nel linguaggio.

---

## 2. Notazione e type system condiviso

### 2.1 Convenzioni

```ml
(* Union chiusa. Nessun costruttore aggiuntivo implicito. *)
type X = A of PayloadA | B of PayloadB

(* Record tecnico, non un oggetto wire. *)
type R = { x : X; y : Y }

(* Tipo nominale: costruttori privati. *)
opaque type Ref<Kind, Signature>
opaque type Proof<Requirement>            (* attestazione di compatibilità governata *)
opaque type RecordAdmissibility<Schema, Operator, Operands>  (* validazione record *)

(* Contenitori con vincoli di costruzione. *)
opaque type AtLeast2<A>
opaque type NonEmpty<A>
opaque type BoundedList<Min, Max, A>
opaque type Nat                     (* intero >= 0 *)
opaque type PositiveInt             (* intero >= 1 *)
opaque type Position1To10
opaque type PositiveDuration
```

`Proof` attesta la compatibilità delle definizioni e dei consumer **governati**, ottenuta risolvendo il registro semantico. `RecordAdmissibility` è invece l’attestazione interna prodotta dalla **validazione schema-aware già esistente del record filtering**: verifica path, tipo della proprietà, operatore e operandi, senza richiedere un registro semantico governato, una `meaningRevision` o nuovi mapping. Nessuna delle due attestazioni è configurabile dall’utente o certifica che i dati di ogni profilo siano completi. Non sono valori pubblici da serializzare né un proof language.

I parametri di tipo descrivono schema, dominio, grana, consumer e ruolo lessicale. Sono indici nominali di prima classe nel modello tecnico, non valori calcolati da espressioni utente. Un'implementazione con tipi meno espressivi può usare descrittori reificati e costruttori privati controllati.

### 2.2 Tipi scalari e quantità

```ml
type ScalarType =
  | Text of TextDomain
  | Number of NumericDomain
  | Boolean
  | Date of DateDomain
  | DateTime of TemporalDomain
  | Enumeration of EnumDefinitionRef
  | Identifier of { domain : IdentifierDomainRef; namespace : NamespaceRef }
  | Money of { domain : MoneyDomainRef; currency : CurrencyRef }
  | Quantity of { domain : QuantityDomainRef; unit : UnitRef }
  | Duration                           (* durata trascorsa, non periodo calendar *)
  | Count of CountMeaning

type CountMeaning =
  | Occurrences of GrainRef
  | DistinctValues of DimensionMeaning
  | Candidates of CandidateGrain

(* CountMeaning è conservato anche quando due count sono confrontabili. *)
type PropertyShape = Scalar of ScalarType | DeclaredSet of ScalarType

type Literal<t> = ExactValue of CanonicalValue<t>
type ReadResult<t> = Known of Value<t> | Missing | ExplicitNull | Unavailable
```

`ReadResult` illustra la distinzione necessaria nell’evidenza; non è un nodo del filtro. **`ExplicitNull` indica, quando proviene da un percorso JSON, un valore presente:** per `RecordCondition` le equality/difference JSON e la membership possono confrontarlo come kind esplicito. Non è automaticamente UNKNOWN; sui confronti non applicabili all’ordinamento può invece produrre UNKNOWN. `Missing` indica il path assente. La rappresentazione concreta dei risultati runtime non viene definita in questa fase.

`Number` è dimensionless. Denaro, quantità, durata e count non sono numeri senza unità. Una durata runtime può essere zero; una durata configurata come ampiezza di finestra o pausa deve essere positiva. Gli offset hanno costruttori distinti `Before Δ | AtAnchor | After Δ`, senza richiedere una durata configurata zero.

Una differenza di count può essere negativa. La soglia di `Difference` usa pertanto `SignedDeltaLiteral<UnitOf<t>>`, non il tipo non negativo del valore `Count`. Questa è una distinzione di parametro, non una nuova riduzione né un nuovo operando disponibile al builder.

### 2.3 Compatibilità: uguaglianza di tipo non basta e non è sempre necessaria

La relazione è specifica dell'operazione:

```ml
opaque type Comparable<A, B>
opaque type OrderedComparable<A, B>
opaque type DifferenceCompatible<A, B, DeltaUnit>
opaque type RatioCompatible<A, B>        (* risultato numerico adimensionale ammesso *)
opaque type SameMeasureUnit<A, B>
opaque type AdditiveField<S, T>
opaque type NonNegativeWeight<S, T>
```

Queste relazioni non sono liberamente dichiarabili nel filtro. Si derivano dai tipi elementari e dai contratti approvati.

| Coppia | Trattamento |
|---|---|
| `Identifier<Product>` / `Identifier<Order>` | Incompatibile, anche se i valori raw sono stringhe |
| Product nello stesso dominio ma namespace diversi | Incompatibile senza una definizione governata che fornisca già l'identità confrontabile |
| `Money<EUR>` / `Money<USD>` | Incompatibile come valori direttamente confrontabili; nessun convertitore inline |
| Denaro / durata | Incompatibile |
| Count ordini / count operazioni Refund | Confronto e ratio ammessi; entrambe le grane restano esplicite |
| Count / durata | Incompatibile; non si costruisce un tasso con unità composta |
| Sum denaro / credit limit corrente nella stessa unità e dominio confrontabile | Ammesso |
| Quantità in scale differenti | Ammesse soltanto se il catalogo dichiara la rappresentazione confrontabile e la normalizzazione esatta pertinente |
| Date / datetime | Nessuna conversione implicita; l'eventuale uso di una Date come ancora richiede DayStart nel calendario dichiarato |
| Campo raw Number / campo raw Number | Percorso typed possibile senza ontology, salvo domini esplicitamente incompatibili |
| Campo raw Text / Identifier governato | Nessuna promozione automatica da stringa a identità business |

La semantica economica del campo — pagato, lordo/netto, rimborso, quota — rimane nel riferimento di proprietà/misura. Avere EUR come valuta non basta a stabilire ogni compatibilità business. D'altra parte, non si impone che Purchase e Refund abbiano lo stesso property ID: la differenza ammessa può confrontare importi di significato economico pertinente ma proprietà differenti.

**Riferimenti normativi (specifica 06 v1.2):** §§6, 8–10, 17, 19, 22.

---

### 2.4 Tipi già presenti nel record filtering

`ScalarType` e `PropertyShape` sopra descrivono gli operandi **semantici** delle nuove costruzioni. Non costituiscono una restrizione retroattiva del catalogo dei campi accettati da `FilterCondition`. In particolare, il record filtering comprende Int e UInt fino a 64 bit, Float con precisione di schema, Decimal esatto, Time, Year, UUID, IP, JSON dinamico, Array typed, Map e Object di schema nei rispettivi operatori ammessi, oltre a semantiche già dichiarate di Money, Measurement e Duration.

```ml
(* Helper chiuso di rappresentazione, non campo pubblico o nuova primitiva. *)
type RecordPropertyShape<S> =
  | OrdinaryTyped of ExistingSchemaType<S>         (* anche Time, Year, Array, Map, Object *)
  | DynamicJSON of ExistingJSONPropertyPath<S>     (* valore a kind runtime *)

opaque type PreparedRecordOperands<Operator, PropertyShape>
```

I parser e i validator esistenti rimangono autorevoli per le combinazioni `operatore × tipo × operandi`. Le soglie del filtro non diventano automaticamente valori da inserire nella colonna: possono superare il range della proprietà nei limiti globali; i Number JSON non devono essere convertiti a Float. Un literal JSON `null` presente resta distinto dal dato assente. Le rappresentazioni esatte e i limiti esistenti non sono adattati automaticamente alle istanze più strette degli operatori semantici nuovi.

Per i nuovi confronti fra proprietà o per misure governate, le regole di compatibilità del modello semantico continuano ad applicarsi; non si usa la presenza di un campo String o Number per inventare identità, unità o domini business.

## 3. Riferimenti semantici stabili e percorso raw

```ml
type SemanticIdentity<K> = {
  namespace : SemanticNamespace;
  definition : DefinitionId<K>;
  meaningRevision : ImmutableMeaningRevision
}

type SemanticRef<K, Signature> = Ref<K, Signature>

type DefinitionKind =
  | ProfilePropertyDefinition | RawBehaviorDefinition | RawPropertyDefinition
  | SemanticEventRole | Repertoire | Grain | SemanticProperty | IdentityRole
  | UnitDefinition | CurrencyDefinition | CalendarDefinition
  | AuthoritativeObjectContract | NamedEvidenceScope
  | AuthorizedMemberSlot | AuthorizedTemporalSlot
  | CertifiedEvidenceDefinition
```

Un riferimento **governato** identifica una definizione e una revisione di significato immutabile. Un ID stabile privo di revisione non sarebbe sufficiente se lo stesso mapping potesse cambiare significato sotto quell’ID. Una revisione immutabile è una scelta tecnica; non si prescrive qui come codificarla sul wire. **La `property` pubblica delle condizioni record esistenti resta invece un path fisico risolto nello schema corrente:** non viene promossa retroattivamente a `SemanticRef` né reinterpretata quando cambiano assegnazioni di ruolo. Il suo raccordo con lo schema è un helper di risoluzione distinto dai contratti semantici governati delle nuove condizioni.

Sono separati:

```ml
type SemanticDefinition<Sig> = {
  identity : SemanticIdentity<DefinitionKind>;
  signature : Sig
}

type DisplayMetadata = { label : Text; description : Optional<Text> }
type PhysicalBinding = { providerBinding : OpaqueImplementationBinding }
```

Display name e raw implementation path non partecipano all'identità semantica. Cambiare la posizione fisica di un campo mantenendone la stessa definizione non richiede di reinterpretare il filtro. Se cambia il significato, serve invece una nuova revisione semantica.

### 3.1 Raw e governato

```ml
type OccurrenceSource<U, S, R> =
  | Raw of RawBehaviorRef<S, R, U>
  | GovernedRole of SemanticRoleRef<S, R, U>
  | GovernedRepertoire of RepertoireRef<S, R, U>
  | CertifiedOccurrence of CertifiedOccurrenceUseRef<S, R, U>
  | CertifiedMilestone of CertifiedMilestoneUseRef<S, R, U>

type FieldRef<S, T> =
  | RawField of RawPropertyRef<S, T>
  | SemanticField of SemanticPropertyRef<S, T>

opaque type GrainProjectionRef<S, G>
opaque type DeduplicationIdentityContract<S, G>
opaque type IdentityRoleAccess<S, G, Domain, Namespace>
opaque type DimensionRef<S, G, T>
```

`U` è uno dei consumer chiusi: per esempio root reduction, selection base, path step, occurrence predicate o activity recognizer. Non è `any`.

Un evento raw tipizzato può essere contato e filtrato senza Purchase ontology. Per presentarlo come Purchase, sommare un “totale pagato” governato o correlare product identity fra repertori occorrono i mapping pertinenti. Un raw path è l'indirizzo fisico di un campo; il suo schema registrato fornisce almeno identità della definizione, tipo e versione.

### 3.2 Source, selector, grain e identity non sono sinonimi

| Componente | Domanda risolta | Esempio |
|---|---|---|
| Source/repertoire | Quali comportamenti costituiscono la popolazione? | Purchase |
| Grain projection | Che cosa costituisce una occurrence della misura? | Ordine; riga; prodotto-per-ordine |
| Dedup identity | Quali record sono copie della stessa occurrence? | business action ID; coppia order/product per la proiezione pertinente |
| Semantic identity role | Che cosa può essere lo stesso oggetto fra evidenze? | product; order |
| Selector | Quale posizione cronologica viene scelta? | Last nella base dichiarata |
| Context restriction | A quale valore locale devono appartenere le occurrence? | Il solo order del candidato K |

L'unione governata di eventi deduplica alla grana dichiarata. Non è una lista arbitraria di subquery e non usa similarità euristiche fra record.

---

## 4. Algebra chiusa dei property predicates

### 4.1 Operatori e accessi

```ml
type EqualityOp = Equal | Different
type OrderedOp = Less | LessOrEqual | Greater | GreaterOrEqual

type Interval<T> = {
  lower : Optional<Bound<T>>;
  upper : Optional<Bound<T>>
}
type Bound<T> = Inclusive of T | Exclusive of T

type ScalarPredicate<S> =
  | RecordCondition of RecordPropertyPredicate<S> (* 26 operatori pubblici condivisi *)
  | EqualLiteral of { field : FieldRef<S, T>; op : EqualityOp; value : Literal<T> }
  | EqualCurrentProfile of {
      field : FieldRef<S, A>; op : EqualityOp;
      current : CurrentProfileScalarRef<B>; compatible : Comparable<A, B>
    }
  | OrderedLiteral of { field : FieldRef<S, T>; op : OrderedOp; value : Literal<T> }
  | OrderedCurrentProfile of {
      field : FieldRef<S, A>; op : OrderedOp;
      current : CurrentProfileScalarRef<B>; compatible : OrderedComparable<A, B>
    }
  | InLiteralSet of { field : FieldRef<S, T>; values : NonEmptySet<Literal<T>> }
  | InLiteralRange of { field : FieldRef<S, T>; range : Interval<Literal<T>> }
  | IsMissing of PropertyRef<S>
  | IsPresent of PropertyRef<S>
  | IsExplicitNull of PropertyRef<S>
  | HasKnownValue of PropertyRef<S>
  | IsEmptyText of FieldRef<S, Text<D>>
  | ContainsText of FieldRef<S, Text<D>> * Literal<Text<D>>
  | StartsWithText of FieldRef<S, Text<D>> * Literal<Text<D>>
  | EndsWithText of FieldRef<S, Text<D>> * Literal<Text<D>>
  | DeclaredSetContainsLiteral of SetFieldRef<S, T> * Literal<T>
  | DeclaredSetContainsCurrent of SetFieldRef<S, A> * CurrentProfileScalarRef<B>
                                  * Comparable<A, B>
  | DeclaredSetContainsOneOf of SetFieldRef<S, T> * NonEmptySet<Literal<T>>
  | MonetaryCurrencyTest of CurrencyBoundPredicate<S>
  | TemporalPropertyTest of TemporalFieldPredicate<S>
```

`T` e gli altri indici sono legati dal costruttore: non sono parametri liberi disaccoppiati dai payload. Gli **operatori governati nuovi** hanno istanze soltanto per tipi ordinabili ammessi e i test testuali governati sono limitati ai field previsti. La variante `RecordCondition` conserva invece l’ordinamento sulle String senza enum e il `contains` sulle forme già ammesse, senza promuovere per questo un Identifier business a testo arbitrario.

Un set dichiarato consente appartenenza esistenziale ai propri elementi, non confronto fra due set, cardinalità di un set arbitrario, iterazioni o traversali. L'appartenenza a un elenco può essere normalizzata a un insieme ordinato di letterali equivalenti. Una lista vuota non introduce un falso costante mascherato.

`IsPresent` riguarda la presenza della proprietà, anche con null; `HasKnownValue` richiede invece un valore noto e valido. `IsExplicitNull` non è `EqualLiteral(null)` e **nessuno di questi tre nodi è un alias obbligatorio degli omonimi operatori record**. Null e missing non appartengono ai normali literal di confronto dei nuovi predicati governati; la membership **record** su JSON continua invece ad ammettere `null` dentro una lista.

Non sono aggiunti regex, codice o negazioni **generiche** di predicate. I 26 operatori record includono già `is not`, `is not between`, `does not contain`, `is not one of`, `is not empty` e `is not null`: sono casi del nodo chiuso `RecordCondition`, non espansioni obbligatorie in gruppi, e non introducono `NOT` su una rule arbitraria. Eventuali nuove riscritture dei predicati governati sono permesse soltanto dopo prova di equivalenza anche su missing e JSON dinamico.

### 4.1.1 Helper di confronto chiusi

```ml
type OrderedOrEqualityLiteralTest<L> =
  | TestLiteral of { operator : EqualityOp | OrderedOp; value : L }
  | TestLiteralInterval of Interval<L>

opaque type NonEmptySet<A>
```

`GreaterThan(x)`, `LessThan(x)`, `AtLeast(x)` e analoghi, dove usati negli esempi, sono abbreviazioni dei corrispondenti operatori di `TestLiteral`. Il tipo del payload deve essere ordinabile per un test ordinato governato; la restrizione non rimuove l’ordinamento su String del **record filtering** attraverso `RecordCondition`. Un range deve avere estremi staticamente compatibili e non essere vuoto per incompatibilità dei parametri.

Le quattro richieste della traccia hanno forme differenti:

```ml
(* Qui amountRaw è dichiarato Number, non denaro privo di unità. *)
OrderedLiteral(event.amountRaw@schema1, Greater, NumberLiteral(100))
EqualCurrentProfile(event.currency@1, Equal, profile.preferredCurrency@1)
IsExplicitNull(event.currency@1)
IsMissing(event.currency@1)
```

La seconda legge il valore corrente del profilo; le ultime due non contengono un operando destro. Un amount monetario usa invece il percorso tipizzato seguente.

### 4.1.2 Condizione record condivisa e valutazione

```ml
(* Nomi tecnici interni: corrispondono 1:1 agli operatori pubblici §8.1 della norma. *)
type ExistingRecordOperator26 =
  | Is | IsNot
  | IsLessThan | IsLessThanOrEqualTo | IsGreaterThan | IsGreaterThanOrEqualTo
  | IsBetween | IsNotBetween
  | Contains | DoesNotContain
  | IsOneOf | IsNotOneOf
  | StartsWith | EndsWith
  | IsBefore | IsOnOrBefore | IsAfter | IsOnOrAfter
  | IsTrue | IsFalse
  | IsEmpty | IsNotEmpty
  | IsNull | IsNotNull
  | Exists | DoesNotExist

type RecordPropertyPredicate<S> = {
  property : ResolvedRecordPath<S>;                 (* path pubblico preservato *)
  operator : ExistingRecordOperator26;              (* enum chiuso dei 26 nomi pubblici *)
  operands : PreparedRecordOperands<operator, property>;
  validity : RecordAdmissibility<S, operator, operands>
}
```

`ExistingRecordOperator26` è un catalogo di costruttori **chiuso** che denota esattamente le 26 denominazioni pubbliche e le combinazioni tipo/cardinalità della specifica normativa 06 v1.2, §8.1. Gli otto operatori unari pubblici richiedono `Values == nil`; `[null]` è un literal JSON Null ammesso solamente dove previsto (membership JSON). Non è una stringa con operatori creati a runtime né una nuova sintassi wire. `RecordPropertyPredicate` si costruisce soltanto dopo risoluzione del percorso e validazione della coppia tipo/operatore/operandi tramite il **validatore record esistente**, attestata da `RecordAdmissibility`: nessuna nuova prova di registry governato è necessaria. Nella forma pubblica resta `FilterCondition {property, operator, values}`. Questa variante è consumabile in V alla root o nei **predicati locali espressamente autorizzati** dai body e dagli `OccurrenceWhere`; un owner non acquisisce operazioni vietate mediante il riuso della variante.

Semantica dei **20** operatori su valori: percorso assente → SQL `NULL` / Go `false` nell'applicabilità, anche sui negativi; JSON di kind presente ma non applicabile ad ordine, range, contains, prefisso e suffisso → `NULL` / `false`. JSON `null` resta presente; uguaglianza/diversità e membership record usano kind distinti e la lista JSON può includere Null. I **sei** operatori espliciti sullo stato del dato conservano i comportamenti dell'implementazione `d69e6168` (§8.2 della norma 06 v1.2). Un filtro invalido o un errore di esecuzione non viene trasformato in un risultato ternario.

La fixture di conformità della specifica 06 v1.2, §28.20, distingue il confronto Record raw `properties.currency is "EUR"` su JSON Null (FALSE) dalla lettura governata `NativeCurrency = EUR` con valuta business sconosciuta (UNKNOWN). Le due costruzioni non sono alias e non possono essere sostituite automaticamente.

La forma canonica è interna e può essere differente dal DTO pubblico; non riinterpreta String literal come raw path o riferimento ad altra proprietà. I nuovi confronti proprietà–proprietà usano costruttori dedicati `EqualCurrentProfile`, `OrderedCurrentProfile` oppure le forme terminali chiuse, senza modificare `RecordPropertyPredicate`.

Per valutazioni comportamentali su più occurrence, il valore incerto deve rimanere distinguibile **fino a quando è stato consumato** da esistenza, assenza, cardinalità, E/K/P/T: l'applicabilità booleana Go di un record non è una prova che un evento mancante dal `WHERE` sia certamente non qualificante. Questa è una regola di composizione dell'evidenza, non un secondo catalogo di operatori.

### 4.2 Valuta dinamica senza spostare i prefiltri

Un importo con valuta per-occurrence è un campo monetario dipendente da un discriminante registrato, non un `Number` convertibile liberamente.

```ml
type CurrencyBoundPredicate<S> = RequireCurrencyAndTest of {
  money : TaggedMoneyFieldRef<S, MoneyDomain>;
  currency : CurrencyRef;
  test : MoneyLiteralPredicate<MoneyDomain, currency>
}
```

Il costruttore ha il significato della congiunzione fra requisito sulla valuta dichiarata e test del valore monetario nella stessa valuta. Il mapping del campo associa il discriminante; non è scelto con un join fra proprietà qualsiasi.

Valuta certamente diversa: requisito NO MATCH. Valuta ignota: nessun valore monetario senza unità viene inventato. Valuta EUR e importo ignoto: test quantitativo UNKNOWN. Questo nodo può stare nel `verify` di E senza filtrare la base; non è un cast e non modifica la selezione.

Un campo già certificato come `paidComparableEUR` è invece direttamente `Money<EUR>`. Non implica che la valuta nativa di ogni ordine sia EUR e non autorizza a scartare gli ordini in altre valute.

### 4.3 Predicati temporali e membro posseduto

Alle occurrence sono disponibili soltanto queste ulteriori foglie temporali:

```ml
type OccurrenceTimePredicate<S> =
  | TimestampIn of ObservationPeriod
  | ElapsedAgeTest of OrderedOrEqualityLiteralTest<PositiveDuration>
  | WithinAfterPersonalAnchor of PersonalAnchor * PositiveDuration
  | CalendarMembership of CalendarMembershipTest<Day | Week | Month>
  | LocalTimeBand of { calendar : CalendarChoice; start : LocalTime; end : LocalTime }
```

Il test di età usa il tempo della stessa occurrence rispetto a T, senza produrre un nuovo operando. `ElapsedAgeTest(GreaterOrEqual, 7 giorni)` include esattamente il bordo dei sette giorni: non è sostituito da un intervallo semiaperto che lo escluda.

```ml
type TemporalFieldPredicate<S> =
  | TemporalFieldIn of TemporalFieldRef<S> * DateValueWindow
  | TemporalFieldAge of TemporalFieldRef<S> * OrderedOrEqualityLiteralTest<PositiveDuration>
  | TemporalFieldWithinAfterPersonalAnchor of
      TemporalFieldRef<S> * PersonalAnchor * PositiveDuration

type DateValueWindow =
  | AbsoluteDateValueInterval of DeclaredTemporalBounds
  | ElapsedPastValueWindow of ClosedElapsedPastBounds
  | UpcomingElapsedValueWindow of PositiveDuration       (* [T,T+d) *)
  | CalendarDateValueWindow of ClosedCalendarRange
  | PersonalDateValueWindow of PersonalWindow
```

Questi sono test diretti su una proprietà temporale o sul momento del ruolo locale. `SelectedMomentTest`, `PivotMomentTest` e i lifecycle test sono le relative specializzazioni autorizzate: stessa appartenenza temporale, confronto di età oppure finestra rispetto a una data personale ammessa. Per una Date si applica la convenzione DayStart dichiarata. La finestra di una proprietà futura non applica il taglio degli eventi a T.

`CalendarMembershipTest` contiene i valori/periodi del catalogo calendar pertinente; non contiene una funzione che produce un numero o una data. La fascia locale mantiene gli estremi dichiarati; per 18:00–22:00 sono `[18:00,22:00)`.

Un where può contenere **una sola occorrenza sintattica** di verifica esistenziale su una lista posseduta:

```ml
type OwnedMemberTest<S> = ExistsOwnedMember of {
  member : OwnedListFieldRef<S, M>;
  predicate : ScalarOnlyBool2<M>
}

type OccurrenceWhere<S> =
  | Unfiltered
  | Where of OneOwnedMemberAtMost<CanonicalBool<2,
        ScalarPredicate<S> | OccurrenceTimePredicate<S> | OwnedMemberTest<S>>>
```

Il payload interno di `OwnedMemberTest` non contiene `OwnedMemberTest`: niente iterazioni annidate. `OneOwnedMemberAtMost` ha costruttori privati con indice 0/1; la composizione somma l'indice e non ammette 2. Consente di collocare quell'unica verifica sotto AND oppure OR, senza imporre una congiunzione aggiuntiva che cambierebbe il significato.

La verifica della riga `categoria=Fotocamera AND quantity>=2` ha un solo membro testimone. Non è una relazione verso un oggetto Product esterno.

**Riferimenti normativi (specifica 06 v1.2):** §§8–9, 10.3, 17, 19.2.

---

## 5. Tempo, calendari e ancore: unioni chiuse

### 5.1 Calendario e periodi non contestuali

```ml
type CalendarChoice =
  | FixedCalendar of CalendarDefinitionRef
  | CurrentProfileTimezone of AuthorizedTimezonePropertyRef * CalendarRulesRef

type ObservationPeriod =
  | AbsoluteHalfOpen of { start : UnambiguousInstant; end : UnambiguousInstant }
  | ElapsedLookback of PositiveDuration                    (* [T-d,T) *)
  | ElapsedAgoBand of { older : PositiveDuration; newer : NonNegativeElapsedOffset }
  | CalendarRange of ClosedCalendarRange
  | CompleteCalendarPeriods of CompleteCalendarSeries
  | FullRelevantHistory of RelationshipHistoryRef
  | AvailableHistory of DeclaredRetentionHistoryRef
  | PersonalAnchored of PersonalWindow

type CompleteCalendarSeries =
  | LastComplete of { kind : Day | Week | Month; count : PositiveInt }
  | ExplicitConsecutiveComplete of {
      kind : Day | Week | Month;
      bounds : CompleteCalendarBounds;
      count : PositiveInt
    }
```

`ClosedCalendarRange` usa soltanto giorno/settimana/mese, estremi calendar espliciti o posizioni calendar costanti rispetto a T. `CompleteCalendarBounds` può identificare una serie delimitata con date calendar oppure estremi calendar relativi a T dichiarati, per esempio le quattro settimane complete precedenti l'ultima settimana completa. La serie è sempre consecutiva, dello stesso tipo e non sovrapposta; non è una lista di bucket selezionati. Il numero M è un parametro noto, non il risultato di una misura.

Le settimane della baseline seguono lunedì–domenica; la scelta del fuso non trasforma questa convenzione in un calendario settimanale arbitrario.

Ogni rule contiene il proprio `CalendarChoice` effettivo. Il filtro conserva inoltre il calendario predefinito esplicito. La canonicalization espande l'ereditarietà della UI: un cambio del default non reinterpreta silenziosamente rule già materializzate.

`FullRelevantHistory(RelationshipHistoryRef)` identifica **tutta la storia pertinente registrata e interrogabile nella vista della query**, fino a T: `RelationshipHistoryRef` rimane il descrittore del repertorio/storia pertinente, non un certificato dell’inizio della relazione reale. `AvailableHistory(DeclaredRetentionHistoryRef)` conserva un intervallo esplicito dichiarato della storia conservata. I due costruttori possono selezionare gli stessi eventi in una vista, ma non vengono fusi automaticamente: un intervallo esplicito e la richiesta di tutta la storia interrogabile restano descrizioni diverse. Nessuna forma attesta eventi cancellati o mai registrati.

Per l’uso event-based 1.2, `RelationshipHistoryRef` si risolve sulle definizioni governate pertinenti della fonte/repertorio già previste in §3, che devono dichiarare il dominio registrato interrogabile. Se una definizione/revisione della 1.1 esprimeva la storia dell’intera relazione reale, non è la stessa definizione semantica: occorre una nuova identità di definizione o una nuova `meaningRevision` immutabile, con risoluzione e adozione esplicita del riferimento. Il solo tag `FullRelevantHistory`, un’etichetta o un alias “latest” non autorizzano a reinterpretare il riferimento precedente. `RegisteredQueryableHistory@1` è l’identificativo illustrativo del descrittore con questo significato, non un nuovo nome assegnato alla stessa identità/revisione della 1.1. Si riusano i riferimenti e le definizioni di fonte/repertorio esistenti, senza nuovi costruttori AST o `DefinitionKind`; le `RecordCondition` conservano il proprio percorso schema-aware senza registry aggiuntivo.

Per `MaximumGap` la forma temporale deve prevedere entrambi gli estremi finiti: `FullRelevantHistory` da solo è invalido; `AvailableHistory` è ammesso soltanto con un intervallo dichiarato completo di estremi. Non si ricava implicitamente l’inizio dal primo evento o da un limite di retention. Questo controllo sul costruttore non restringe gli altri usi di FullRelevantHistory (§§13.3, 15).

First lifetime, “mai” e novità si interpretano su questo dominio. Non occorrono capacità di certificare un prefisso/coda esterni o l’inizio della relazione reale. Retention può modificare il primo evento identificabile, l’ordine degli inizi registrati delle opportunità e le pause. Dati ignoti di eventi esistenti restano pertinenti. I timestamp di lifecycle autorevoli e i fatti di first-attainment certificati conservano invece le proprie garanzie. Le forme ADT non cambiano e non nasce una nuova modalità di selezione (§§8–11 e 06 §§3.1, 9.4, 19).

### 5.2 Ancore personali e finestre locali

```ml
type PersonalAnchor =
  | CurrentDateTimeAnchor of AuthorizedProfileDateTimeRef
  | CurrentDateAtDayStart of AuthorizedProfileDateRef * CalendarChoice

type PersonalWindow =
  | BeforePersonalDate of PersonalAnchor * PositiveDuration   (* [a-d,a) *)
  | WithinAfterPersonalDate of PersonalAnchor * PositiveDuration
                                                               (* [a,a+d], deadline inclusa *)
  | PersonalDateUntilT of PersonalAnchor                       (* [a,T) *)
  | BetweenDeclaredPersonalDates of AuthorizedPersonalIntervalRef
  | DeclaredPersonalOffsetInterval of {
      anchor : PersonalAnchor; bounds : ConstantOffsetBounds
    }

type OccurrenceWindow<Owner> =
  | BeforeSelfWithin of PositiveDuration
  | AfterSelfWithin of PositiveDuration
  | AfterSelfUntilT

type LifecycleWindow<Contract, Slot> =
  | DeclaredLifecycleHorizon of AuthorizedLifecycleHorizonRef<Contract, Slot>
```

`ConstantOffsetBounds` contiene soltanto offset costanti e il bordo ammesso della forma; non può combinare due timestamp locali scelti liberamente. `BetweenDeclaredPersonalDates` è il periodo personale tipizzato già ammesso dal catalogo, non una formula fra date arbitrarie. Per un ciclo corrente la forma canonica è `PersonalDateUntilT(cycleStart)`.

Le ancore locali sono **ruoli distinti**, non un tipo universale `Reference<DateTime>`:

```ml
(* Tag interni, introdotti esclusivamente dal rispettivo owner. *)
type LocalAnchorRole =
  | SelectedOccurrenceSelf
  | CorrelatedPrimaryPivotSelf
  | OpportunityStartSelf
  | PathAdjacentEndpoints
  | PathFirstAndLast
  | PathFinalSelf
  | AuthoritativeLifecycleSelf<Contract, Field>
  | QualifiedReturnSelf
```

Questa enumerazione documenta i ruoli; non viene esposta come un'unione liberamente selezionabile negli slot. Ogni slot accetta soltanto il proprio sottotipo. Una milestone selezionata in E usa l'ancora della selected occurrence di quell'E; una milestone in P usa lo slot del proprio passo, mai una nuova ancora globale.

### 5.3 Bordi derivabili dal costruttore

| Costruttore | Regola obbligatoria |
|---|---|
| Absolute / elapsed / calendar | Intervallo `[start,end)`; gli eventi usati sono sempre precedenti a T |
| Prima di data personale | `[a−Δ,a)` |
| Prima di occurrence | `e ≺ a` e `time(a)−time(e) <= Δ`; a esclusa; ordine affidabile può distinguere timestamp uguali |
| Dopo occurrence entro Δ | `a ≺ e`, ritardo `<= Δ`; deadline inclusa |
| Dopo occurrence fino a T | `a ≺ e` e `time(e)<T`; nessuna deadline fissa ulteriore |
| Fra passi | Stretta intermedietà nell'ordine effettivo, non solo timestamp |
| Primi Δ del trial | `[start,start+Δ)` e assegnazione al trial quando richiesta |
| Human Response dalla prima apertura | `[openedAt,T)`; stessa identity/lifecycle; eventuale ordine iniziale certificato rispettato |
| Rolling burst | `[s,s+Δ)`, interamente nel periodo esterno |
| Proprietà data nei prossimi Δ | `[T,T+Δ)`; non implica usare eventi futuri |

Una deadline negativa `d` è matura solo con **T > d** e valutazione delle occurrence registrate pertinenti fino a d inclusa, mantenendo le eventuali garanzie specifiche del contratto; non serve un’attestazione degli eventi reali esterni. `T=d` non basta. Una finestra futura non è accorciata a T. Una prova positiva può decidere un requisito monotono prima della maturazione; la mancanza di prova non diventa un'assenza.

Per timestamp di lifecycle si usa il tipo autorizzato dal contratto: una data non acquisisce per magia l'ordine di un'occurrence. Date locali ambigue o inesistenti richiedono una disambiguazione esplicita prevista dal calendario; timezone del profilo mancante non usa il workspace come fallback.

**Riferimenti normativi (specifica 06 v1.2):** §§9.4–9.5, 13.5, 14, 15.6–15.7, 17.

---

## 6. Behavior Scope, ownership e budget strutturali

### 6.1 Il componente condiviso

Behavior Scope è un helper canonico condiviso, non una primitiva e non una query riutilizzabile dall'utente.

```ml
type BehaviorPopulation<U, S, R, G> = {
  source : OccurrenceSource<U, S, R>;
  grain : GrainProjectionRef<S, G>;
  where : OccurrenceWhere<S>
}

type FreeBehaviorScope<U> = {
  population : BehaviorPopulation<U, S, R, G>;
  period : ObservationPeriod
}

type RelativeBehaviorScope<Owner> = {
  population : BehaviorPopulation<RelativeLeaf, S, R, G>;
  window : OccurrenceWindow<Owner>;
  additionalPeriod : Optional<ObservationPeriod>
}
```

`additionalPeriod` è un'ulteriore restrizione esplicita, non un periodo che tronca implicitamente la deadline. La finestra autorizzata dall'owner continua a determinare ordine e maturazione. Non è disponibile una union/intersezione arbitraria di espressioni temporali.

Gli altri scope sono specializzazioni **diverse**, con l'identità incorporata nel ruolo:

```ml
opaque type ObservedValueScope<Owner, Repertoire, Dimension>
opaque type SameIdentityScope<Owner, IdentityRole>
opaque type OpportunityOutcomeScope<Owner, OpportunityKind>
opaque type NamedObjectEvidenceScope<Owner, Contract, NamedSlot, Capabilities>
opaque type CartProductBehaviorScope<Owner, MemberOwner>
opaque type TrialUpgradeTemporalScope<Owner>
opaque type PathBetweenScope<Owner, KeyVisibility>
opaque type PathFinalOutcomeScope<Owner, KeyVisibility>
```

La parte comune è esattamente popolazione tipizzata, grana, where same-occurrence, periodo e restrizioni fornite dal proprietario. Le specializzazioni non aggiungono subquery, risultati aggregati nel where o arbitrary timestamp expression.

Un `SameIdentityScope` non ha un parametro `keyValue`: il valore è quello dell'owner. Un `ObservedValueScope` non ha un parametro per cambiare repertorio o dimensione. Un `NamedObjectEvidenceScope` non ha un source selector raw sostituibile. Un `TrialUpgradeTemporalScope` non ha una chiave trial.

### 6.2 Ownership scelta: struttura prima, token nominali soltanto interni

Ogni costruzione contestuale introduce un owner fresco `o`. Gli accessi comuni sono impliciti: `SelfProperty`, `BeforeSelfWithin`, `AfterSelfUntilT`, `SameCandidateIdentity`. Non sono necessari ID di occurrence o nomi di variabili nell'AST.

```ml
opaque type OwnerToken<Role>          (* generativo, non scelto dall'utente *)
opaque type LocalScalar<O, Role, T>   (* costruttori disponibili solo nell'owner *)

(* Schematizzazione della chiusura, non un nuovo nodo configurabile. *)
type ClosedOwnerConstruction<Role, Payload> =
  PackOwner of exists O. { payload : Payload<O> }
```

`exists O` nasconde l'owner alla rule sorella. Due copie di una rule hanno owner distinti, anche se strutturalmente equivalenti. I token non partecipano alla canonical equality: si confrontano i ruoli strutturali, modulo rinomina del token tecnico.

La regola di scope è:

```text
Γ = { CurrentProfile, EvaluationTime, EffectiveCalendar, SemanticDefinitions }

Γ ; O:Role ; AuthorizedSlots(Role) ⊢ body : Truth
─────────────────────────────────────────────────────────────
Γ ⊢ CloseOwner(Role, body) : Rule

Nella conclusione non compaiono O, LocalScalar<O,...> o Anchor<O,...>.
Per ogni sibling si riparte dallo stesso Γ, senza i binding delle altre sibling.
```

Il solo accesso aggiuntivo a membri è:

```text
CartOwner(o) → CartProductMemberOwner(m, parent=o) → leaf behavior
```

Il membro non introduce altri membri. La verifica di una riga già posseduta da una occurrence è invece un predicato same-evidence, non una traversale di oggetti.

### 6.3 Firme locali, non un universal context

| Owner | Scalars locali leggibili | Scope ammessi | Costruzioni contestuali nel body |
|---|---|---|---|
| Root | Nessuno | Free behavior scope | Soltanto le cinque rule, combinate alla root |
| E single | Proprietà e momento della selected occurrence | Relative leaf prima/dopo Self | Un solo FirstFollowing chiuso; nessun E/K/P/T |
| E latest N | Nessun accesso indicizzato libero | Nessun nuovo scope per ogni posizione | Solo predicato uniforme o preset finito |
| K observed value | Valore della dimensione candidata | Stesso repertorio e stessa dimensione vincolata | Nessuno |
| K correlated object | Identity e proprietà primarie autorizzate; pivot se dichiarato | Tutti sulla stessa identity; finestre proprie o del pivot | Nessuno |
| K opportunity | Proprietà dell'opportunità e start | Outcome attribuito all'opportunità | Nessuno |
| K state-only object | Sole proprietà correnti autorizzate | Nessuno | Nessuno |
| K named-evidence object | Proprietà/lifecycle autorizzati | Soli nomi del contratto e operazioni dichiarate | Nessuno |
| K Cart | Proprietà/lifecycle correnti del cart | Solo tramite il proprio prodotto membro | Solo il member slot CartProducts |
| K Trial | Proprietà/lifecycle del trial designato | Usage attribuito al trial; Upgrade temporale fisso | Nessuno |
| P | Predicati del singolo passo, non scalar access fra passi | Intervalli adiacenti, primo–ultimo, esito finale | Nessuno |
| T last qualified return | Proprietà del ritorno e suo momento | Corpo foglia di esperienza singola | Solo gli slot chiusi di quel corpo |

“Proprietà autorizzate” non significa tutte le proprietà di qualunque oggetto con la stessa identity. I riferimenti portano schema, owner e slot. I test locali riusano i componenti di V solo nei limiti delle firme sopra: non accettano l'intera top-level `Rule`.

### 6.3.1 Espansione delle famiglie di tipo locali

I nomi `AllowedLeafScope<C>`, `LocalScalarAccess<C,T>` e le corrispondenti firme dei test sono **famiglie di tipo chiuse**, non interfacce aperte implementabili da un plugin utente:

| C | AllowedLeafScope<C> | LocalScalarAccess<C,T> |
|---|---|---|
| Root | FreeBehaviorScope<RootReductionUse> | Never |
| Selected / QualifiedReturn | RelativeBehaviorScope del proprio Self | Proprietà/momento della occurrence selezionata o del ritorno, nei ruoli ammessi |
| ObservedValue<R,D> | ObservedValueScope del medesimo R e D | Valore della dimensione candidata D |
| CorrelatedObject<K> | SameIdentityScope della sola K | Proprietà primarie/pivot soltanto se PrimaryAccess le introduce |
| NamedEvidenceObject<C> | NamedObjectEvidenceScope del contratto C | Proprietà/lifecycle di C ammessi alla lettura |
| CartProduct<O,M> | CartProductBehaviorScope del proprio prodotto membro | Sole proprietà del membro contenuto esposte dal contratto, non stato di un altro oggetto Product |
| Trial<O> | TrialAttributedUsageScope oppure TrialUpgradeTemporalScope | Sole proprietà/lifecycle del trial corrente designato |
| Opportunity, Path, StateOnly, Cart root | Never per i test V comportamentali generici | Soltanto le letture delle rispettive leaf union; non un LocalV body universale |

`LocalVTest<C>` istanzia i test chiusi di V con queste sole letture/sorgenti. I campi, le dimensioni, gli operatori e i reducer sono inoltre intersecati con le capability dello slot, quando governate da un contratto. Le foglie di Opportunity e P mantengono i propri outcome/negative test chiusi, non istanziano liberamente LocalVTest.

Una distribuzione ammessa come confronto tipizzato locale usa esclusivamente lo scope della propria riga e una dimensione dichiarata non esportabile. Nella forma ObservedValue la dimensione di distribuzione, se consumata, è soltanto D già vincolata: non crea una seconda distribuzione su una dimensione arbitraria. Le diverse dimensioni espressamente ammesse per **CountDistinct**, come Order o CalendarDay, rimangono il distinto slot di riduzione atomica, non questa apertura di una nuova distribuzione. In uno scope di contratto non si inferisce il permesso di un test dalla sola presenza dei suoi campi: occorre il relativo tag di operazione già ammesso e dichiarato; Human Response baseline non ottiene automaticamente nuove operazioni.

### 6.4 Budget: parte della grammatica finita

Si impiegano indici di risorsa piccoli, non un dependent type system generale:

```ml
opaque type ClosedBody<C, MaxDepth=2, MaxScopes=4, MaxFollowing=1, MaxMember=1>
```

Ogni foglia ha un costo strutturale determinato dalla sua forma. Un costruttore AND/OR compone soltanto payload il cui totale rientra nei limiti. Un builder può implementare questi indici con contatori controllati e costruttori privati: non esiste un `AnyBody` valido da vietare a posteriori.

Regole di conteggio:

- Un uso di scope subordinato costa 1. Due operandi che incorporano due scope costano 2, anche se gli scope sono semanticamente identici.
- La guardia sullo stesso scope non costa un nuovo posto. La part interna di una share non costa un secondo scope.
- Il FirstFollowing costa uno scope. I predicati sul seguito non aprono ulteriori scope.
- Il seme di K, la base selettiva di E e i passi positivi propri di P non sono scope subordinati. Gli scope negativi/finali di P lo sono.
- Il member slot del Cart non azzera il budget: le sue foglie consumano i quattro posti del Cart proprietario.
- I predicati scalari e la verifica same-evidence su una riga non sono nuovi behavior scopes.
- Il conteggio segue le **posizioni semantiche d'uso**, non il numero di oggetti in memoria, il numero di hash diversi o una condivisione del piano di esecuzione.

La profondità booleana è calcolata nell’algebra pertinente. Per il filtro principale: massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. Il body contestuale e il where same-occurrence mantengono il massimo di due livelli. Una `FilterRule` Behavioral conta come un nodo principale, indipendentemente dalla complessità del proprio body, che mantiene budget autonomi. La validazione della struttura pubblica precede la normalizzazione. Una formula sulle proprietà del medesimo Self viene inserita nel body, non nascosta in una foglia opaca per aggirarne il limite. Un where di uno scope di riduzione riguarda invece un'altra popolazione e ha la propria algebra same-occurrence limitata.

**Riferimenti normativi (specifica 06 v1.2):** §§7, 9, 15.

---

## 7. V — Operandi atomici, riduzioni, guardie e terminali

### 7.1 La barriera di tipo

```ml
type ReadableAtom<C, T> =
  | CurrentProfile of CurrentProfileScalarRef<T>
  | CertifiedCurrentScalar of CertifiedScalarUseRef<T, CurrentValueSlot<C>>
  | AuthorizedLocal of LocalScalarAccess<C, T>
  | Reduced of AtomicReduction<AllowedLeafScope<C>, T>

type AtomicOperand<C, T> =
  | Read of ReadableAtom<C, T>
  | LiteralParameter of Literal<T>

(* Non contiene TerminalTest, V, Rule, Expression o producer invocation. *)
```

I letterali sono classificati come parametri atomici, ma gli slot sono più stretti del tipo complessivo. Un confronto diretto ha almeno un valore letto; i due input delle forme difference/ratio/relative change sono valori atomici letti, mentre la soglia è costante. Non viene introdotta una condizione “due costanti a confronto”, né una formula composta da letterali.

```ml
type DirectComparison<C> = Compare of {
  left : ReadableAtom<C, A>;
  right : ReadableAtom<C, B> | Literal<B>;
  operator : TypedComparisonOperator<A, B>;
  compatibility : ComparisonCompatibility<A, B>
}

type TerminalTest<C> =
  | Direct of DirectComparison<C>
  | Difference of {
      left : ReadableAtom<C, A>; right : ReadableAtom<C, B>;
      compatibility : DifferenceCompatible<A, B, U>;
      test : OrderedOrEqualityLiteralTest<SignedDeltaLiteral<U>>
    }
  | Ratio of {
      numerator : ReadableAtom<C, A>; denominator : ReadableAtom<C, B>;
      compatibility : RatioCompatible<A, B>;
      test : OrderedOrEqualityLiteralTest<DimensionlessLiteral>
    }
  | RelativeChange of {
      recent : ReadableAtom<C, A>; baseline : ReadableAtom<C, B>;
      compatibility : SameMeasureUnit<A, B>;
      test : OrderedOrEqualityLiteralTest<SignedFractionLiteral>
    }
  | Share of OccurrenceShareTest<C>
```

`Ratio`, `RelativeChange` e share portano intrinsecamente la guardia di denominatore positivo. Non hanno un flag per disattivarla. Difference non ha tale guardia.

`TerminalTest` conclude in Truth. Nessun suo costruttore appartiene a `ReadableAtom`. Pertanto `ratio(ratio(A,B),C)`, `difference(A,B)/C`, `(A+B)/C`, `min(A)+max(B)` e `ratio(A,B)>ratio(C,D)` non hanno una derivazione di tipo.

### 7.2 Riduzioni atomiche

```ml
type OptionalSampleMinimum = NoExtraMinimum | AtLeastOccurrences of PositiveInt

type AtomicReduction<L, T> =
  | Count of { scope : L; sample : OptionalSampleMinimum }
      -> Count<OccurrenceGrain<L>>
  | Sum of {
      scope : L; field : AdditiveQuantitativeField<L, T>;
      sample : OptionalSampleMinimum
    } -> T
  | Average of {
      scope : L; field : AverageableQuantitativeField<L, T>;
      minimumOccurrences : PositiveInt
    } -> T
  | Minimum of {
      scope : L; field : NumericOrTemporalField<L, T>;
      minimumOccurrences : PositiveInt
    } -> T
  | Maximum of {
      scope : L; field : NumericOrTemporalField<L, T>;
      minimumOccurrences : PositiveInt
    } -> T
  | CountDistinct of {
      scope : L; dimension : AuthorizedDistinctDimension<L, D>;
      sample : OptionalSampleMinimum
    } -> Count<DistinctValues<D>>
```

La notazione `-> T` in questi costruttori indica il tipo di risultato associato del GADT, non una funzione configurabile. `scope` è sempre una foglia; `field` e `dimension` non accettano `ReadableAtom` o risultati di altre riduzioni.

`AuthorizedDistinctDimension` è un campo/proiezione dichiarata o il giorno calendar dell'occurrence nella forma chiusa ammessa. Per K prodotto → ordini distinti si usa la dimensione order; per K canale → giorni distinti si usa il giorno calendar. Non si costruisce una chiave con una formula.

| Riduzione | Input/output e grana | Vuoto certo | Guardie | Regole di dominio/evidenza |
|---|---|---|---|---|
| Count | Occurrence distinte alla grana scelta → Count di quella grana | 0 | Nessuna obbligatoria; minimo esplicito se richiesto | Dedup identity affidabile; possibili occurrence non diventano scarti |
| Sum | Campo quantitativo additivo per occurrence → stessa unità | 0 nella stessa unità | Minimo campione opzionale; dominio quantitativo valido | Non somma importi di valute non confrontabili; null non è zero; segno certificato governa eventuali limiti |
| Average | Campo quantitativo / popolazione del medesimo scope → stessa unità | Non definita; minimo fallisce | Minimo >=1 obbligatorio | Popolazione e valori adeguatamente noti; non media dei soli non-null |
| Min/max | Campo numerico, quantità, data/datetime → stesso tipo | Non definito; minimo fallisce | Minimo >=1 | Nessun sentinella; prove unilaterali ammesse secondo §19 |
| Count distinct | Una dimensione dichiarata → Count di valori distinti | 0 | Nessuna minima obbligatoria | Ignoto non è categoria null; dominio e uguaglianza della dimensione sono stabili |

La guardia campione associata a una riduzione è un requisito del test che la consuma, non una trasformazione del valore numerico in un valore sostitutivo. `Count(scope, minimum=10)` continua a indicare il count dello scope; il confronto che lo usa richiede anche almeno 10 occurrence.

### 7.3 Share part/whole

```ml
type OccurrenceShareTest<C> =
  | FrequencyShare of {
      whole : AllowedLeafScope<C>;
      part : AdditionalSameOccurrencePredicate<whole>;
      minimumOccurrences : PositiveInt;
      test : OrderedOrEqualityLiteralTest<Fraction0To1>
    }
  | WeightedShare of {
      whole : AllowedLeafScope<C>;
      part : AdditionalSameOccurrencePredicate<whole>;
      weight : NonNegativeAdditiveWeight<whole, W>;
      minimumOccurrences : PositiveInt;
      minimumWholeWeight : Optional<PositiveLiteral<W>>;
      test : OrderedOrEqualityLiteralTest<Fraction0To1>
    }
```

Il costruttore non contiene un secondo scope, un secondo periodo o una seconda grana. `AdditionalSameOccurrencePredicate<whole>` è il where aggiuntivo con il budget residuo di verifiche owned-member: whole e part non possono introdurre due verifiche indipendenti su membri della stessa occurrence aggirando il limite del where effettivo. La part restringe solo la qualificazione della stessa popolazione whole. I pesi della part sono lo stesso campo della whole, non una seconda misura. Le occurrence con esito part UNKNOWN restano nel denominatore.

La **share di candidati** è invece un quantificatore K (§9), applicato a quella candidate base e al body autorizzato. Non si aggiunge `V.Share(KQuery)`: sarebbe un annidamento contestuale non necessario. Si riusa soltanto il helper tecnico `CandidateShareParameters`.

### 7.4 Distribuzioni chiuse e prevalenza

Queste forme completano V senza ampliare gli operandi:

```ml
type FrequencyDistribution<C, D> = {
  scope : DistributionScopeAuthorizedIn<C>;
  dimension : AuthorizedDistributionDimension<C, D>;
  minimumOccurrences : PositiveInt
}

type WinnerValueTarget<C, D> =
  | LiteralValue of Literal<D>
  | CurrentValue of AuthorizedCurrentScalarRef<D>
  | LocalValue of AuthorizedLocalValueTarget<C, D>

type DistributionTerminal<C> =
  | UniqueWinnerTest of {
      distribution : FrequencyDistribution<C, D>;
      operator : EqualityOp;
      target : WinnerValueTarget<C, D>
    }
  | WinnerChangeTest of {
      population : FixedDistributionPopulation<C, D>;
      previousPeriod : AuthorizedDistributionPeriod<C>;
      recentPeriod : AuthorizedDistributionPeriod<C>;
      previousMinimum : PositiveInt;
      recentMinimum : PositiveInt;
      check : CompareWinners of EqualityOp
            | VerifyTwoValues of { previous : Literal<D>; recent : Literal<D> }
    }
  | MaximumValueShareTest of {
      distribution : FrequencyDistribution<C, D>;
      maximumAllowedShare : Fraction0To1
    }
```

Il vincitore resta interno al costruttore. La guardia di unicità vale anche per `Different`: una parità nota non dimostra “un vincitore diverso da Scienza”. La quota massima usa tutte le categorie effettive della stessa distribuzione; non un elenco scelto dall'utente. La prevalenza baseline è per frequenza, non ponderata.

Per winner, confronto fra winner e quota massima, UNKNOWN dipende dalla possibilità che i dati ignoti cambino la **verità del test completo richiesto, guardie incluse**, non soltanto l’identità di un possibile vincitore. Un test certamente falso rimane NO_MATCH anche se l’identità o l’unicità del vincitore non è determinabile. Con Scienza=0, Arte=10, Tecnologia=10 e una occurrence di categoria ignota, `UniqueWinnerTest(Equal, Scienza)` è NO_MATCH: Scienza arriva al massimo a 1. Con gli stessi dati `Different Scienza` è UNKNOWN, perché la guardia di unicità può essere soddisfatta oppure no (06 §§10.4, 19.4, 28.25). Massa ignota, minimo campione e assenza di tie-break restano obbligatori.

`MaximumValueShareTest` è la forma normativa “nessun valore supera X%”, quindi il confronto interno è `maxShare <= X`. Non si espone `maxShare` come scalare.

L'istanza root di queste forme è sempre disponibile con gli scope ammessi. Un uso locale è ammesso solo quando la firma del body autorizza quel test e il suo scope: non si deriva dal mero possesso di un owner. In particolare, la forma K observed-value non acquisisce una seconda dimensione navigabile, e un named scope non acquisisce operazioni non dichiarate dal suo contratto. I target locali esistono solo se quel ruolo autorizza già la lettura del valore scalare; non sono i vincitori di un'altra distribuzione.

### 7.5 Modello delle guardie

Non esiste `GuardExpression`. Le guardie sono obblighi associati a costruttori chiusi:

| Guardia | Dove vive | Parametro | Risoluzione |
|---|---|---|---|
| Minimum sample | Count/sum quando esplicito; avg/min/max; distribuzione/share | N positivo | Consistenza statica; soddisfazione runtime tramite evidenza sullo stesso scope |
| Denominator positive | Ratio, relative change, share | Intrinseco; eventuale minimo aggiuntivo tipizzato | Può essere derivabile da limiti certificati; altrimenti runtime |
| Non-empty candidate set | K All e K Share | Intrinseco; All può richiedere minimo N | Runtime; insieme vuoto certo fa fallire |
| Required N occurrences | E nth/latest N/preset e latest N opportunities | N 1–10; preset 2 o 3 fisso | Statico per il limite; runtime per esistenza e identificabilità |
| Valid quantitative domain | Lettura quantitativa, somma, ratio, peso share | Nessun bypass utente | Unità/dominio statici; dati e validità runtime |
| Unique winner | Ogni winner test | Intrinseco | Unicità e osservazione runtime |
| Positive two intervals | Preset E ultime tre occurrence | Intrinseco, entrambi positivi | Runtime |
| Mature negative deadline | Esito negativo Within | Intrinseco | T>deadline e dati registrati pertinenti fino alla deadline inclusa; garanzie specifiche del contratto quando richieste |

Le guardie si compongono con il test mediante la logica ternaria normativa. Una guardia certamente falsa determina NO MATCH anche se il valore aritmetico non è definito. Una guardia ignota non viene sostituita con true. Le verifiche non invocano aritmetica su un valore non definito per produrre un errore numerico artificiale.

**Riferimenti normativi (specifica 06 v1.2):** §§10, 12.3, 19.

---

## 8. E — Base → select → verify

### 8.1 Esperienza singola e blocco latest N

```ml
type SelectionBase<S> = FreeBehaviorScope<SelectionBaseUse>

type SingleSelector = First | Last | Nth of Position1To10

type SelectedExperience =
  | SingleExperience of exists O. {
      base : SelectionBase<S>;
      select : SingleSelector;
      verify : SingleExperienceBody<O, S>
    }
  | LatestNPredicate of {
      base : SelectionBase<S>;
      size : Position1To10;
      predicate : OccurrenceWhere<S>;
      quantifier : LatestBlockQuantifier
    }
  | LastVersusPrevious of {
      base : SelectionBase<S>;
      field : PairComparableFieldRef<S, T>;
      check : LastPreviousCheck<T>
    }
  | LastTwoIntervals of {
      base : SelectionBase<S>;
      ratioTest : OrderedOrEqualityLiteralTest<DimensionlessLiteral>
    }

type LatestBlockQuantifier = Some | All | Cardinality of CardinalityTest

type LastPreviousCheck<T> =
  | SameValue | DifferentValue
  | NumericDifference of OrderedOrEqualityLiteralTest<SignedDeltaLiteral<UnitOf<T>>>
  | PositiveDenominatorRatio of OrderedOrEqualityLiteralTest<DimensionlessLiteral>
```

`NumericDifference` e `PositiveDenominatorRatio` hanno istanze soltanto per campi quantitativi compatibili. Il field è **uno**, letto sulle ultime due occurrence. Non esistono due field reference né due indici liberamente configurabili. Le posizioni sono esattamente Last e Previous. Il preset non include un operatore generico di relative change; una presentazione equivalente può usare la ratio con soglia trasformata come costante, nei casi ammessi.

`LastTwoIntervals` seleziona sempre le ultime tre occurrence e legge soltanto le due durate native fra coppie consecutive. Entrambe devono essere positive. Non offre accesso agli estremi come nuove ancore o alle durate come operandi esterni.

### 8.2 Normalizzazione dei nomi dei selettori

| Nome prodotto | Forma canonica |
|---|---|
| First lifetime | `base.period=FullRelevantHistory`, where appropriato, `select=First` |
| First in base | Periodo esplicito della base, `First` |
| First qualifying | Prefiltro nella base, `First` |
| Last | Base del repertorio fino a T senza prefiltri aggiunti, `Last` |
| Last qualifying | Prefiltro nella base, `Last` |
| Nth | `Nth N`; N=1 può normalizzare a First |
| Latest N | Costruttore distinto LatestNPredicate o preset fisso |

Non occorre un flag `qualifying`: è il where **della base** a conservare il significato. La normalizzazione riduce la duplicazione dei selettori, non sposta predicate attraverso il confine di selezione.

First e Last si applicano alla base di eventi registrati interrogabili. Un prefisso antico non conservato non rende First UNKNOWN per la sola possibilità di eventi reali antecedenti: si sceglie il primo registrato identificabile, senza pretendere il primo della vita reale. Analogamente Last non richiede una certificazione della coda esterna. Rimangono decisivi qualificazione, timestamp, identity/deduplicazione e ordine di occurrence esistenti: un qualificante precedente o successivo possibile può cambiare la selezione. Non si promuove verify a prefiltro e non si sostituisce l’ultimo con l’ultimo non-null.

### 8.3 Body singolo e first-following

```ml
type SingleExperienceLeaf<O, S> =
  | SelectedScalarPredicate of ScalarPredicate<S>
  | SelectedTimestampPredicate of SelectedMomentTest<O>
  | SelectedOwnedMemberPredicate of OwnedMemberTest<S>
  | AuthorizedValueTest of LocalVTest<SelectedOwner<O, S>>
  | RelativeOutcome of ClosedOutcomeTest<RelativeBehaviorScope<O>>
  | FirstFollowing of FirstFollowingSlot<O>

type SingleExperienceBody<O, S> =
  | NoAdditionalVerification
  | Verify of ClosedBody<SingleExperienceLeaf<O, S>>

type FirstFollowingSlot<O> = {
  source : BehaviorSourceAtGrain<FollowingUse>;
  verify : FollowingOccurrencePropertyAndDelayBody
}
```

Il FirstFollowing cerca la prima occurrence di quel comportamento effettivamente successiva a Self, fino a T. Il suo body contiene solo predicati sul seguito e confronti del ritardo rispetto a Self. Non ha una seconda `RelativeBehaviorScope`, un nuovo FirstFollowing o un selector personalizzabile. Eventuali qualificazioni che definiscono il comportamento sono parte della sorgente/repertorio dichiarato; i requisiti sul seguito scelto rimangono in `verify`.

Se non esiste il seguito, la condizione di “primo contatto tardivo” non è soddisfatta: assenza non significa ritardo infinito. Il limite di una sola posizione FirstFollowing è imposto dal budget del body, anche sotto OR.

`NoAdditionalVerification` è il body vuoto di requisiti aggiuntivi dopo una selezione richiesta, non un gruppo AND vuoto. La costruzione continua a richiedere l'esistenza e l'identificabilità dell'esperienza.

### 8.4 Nessun fallback e nessuna posizione libera

La selezione avviene prima del body. Un SelectedScalarPredicate falso non autorizza a provare un'altra occurrence. Una selezione con più ordini compatibili può avere un risultato determinabile se tutti gli esiti possibili coincidono; il modello non inventa un tie-break tecnico. Latest N deve contenere N occurrence, non “fino a N”.

Il riconoscimento di un ultimo ritorno qualificante è una variante chiusa di T (§13) che riusa questo body tecnico. Non è una composizione generale `E(T(...))` e non aggiunge una fonte query-valued a SelectionBase.

**Riferimenti normativi (specifica 06 v1.2):** §§7.2, 11, 14.5, 17.

---

## 9. K — Discriminated union di insiemi con corpi diversi

### 9.1 Quantificatori condivisi, non corpi condivisi

```ml
type CardinalityTest =
  | AtLeast of Nat
  | AtMost of Nat
  | Exactly of Nat
  | BetweenInclusive of { lower : Nat; upper : Nat }

type CandidateQuantifier<G> =
  | Some
  | None
  | All of { minimumCandidates : PositiveInt }
  | Cardinality of CardinalityTest
  | Share of {
      minimumCandidates : PositiveInt;
      test : OrderedOrEqualityLiteralTest<Fraction0To1>
    }
```

`G` documenta la candidate grain: valore osservato, product identity, order identity, opportunità message/campaign, ticket lifecycle, cart lifecycle, subscription, trial. Non è la grana delle occurrence nel body.

La quantificazione generale di §12.3 è condivisibile dalle quattro famiglie K: non autorizza a condividere i body. Le limitazioni ulteriori derivano dalla cardinalità della base: un trial designato ha al massimo un candidato; un blocco latest N opportunities, se identificabile, ne ha esattamente N. Parametri incompatibili con quei vincoli sono invalidi.

Latest N di **E** usa invece `LatestBlockQuantifier`, più stretto: non eredita automaticamente share o All con un minimo indipendente. Eventuali cardinalità zero del blocco riguardano quante delle N occurrence soddisfano il predicato, non cancellano il requisito che N occurrence esistano.

| Quantificatore K | Base certamente vuota |
|---|---|
| Some | NO MATCH |
| None | MATCH |
| Cardinality | Confronta 0 con la soglia |
| All(min>=1) | NO MATCH |
| Share(min>=1) | NO MATCH per guardia, non quota artificiale 0 |

I candidati con body UNKNOWN restano candidati, anche nel denominatore. Un candidato con body MATCH può provare Some; uno con body NO_MATCH può confutare All. Non si definisce None come un generic NOT di una query.

**Quantificazione composizionale approvata N2, obbligatoria per tutti gli executor.** Con base determinata di D candidati, S body MATCH, F NO_MATCH e U UNKNOWN, S+F+U=D e l’inviluppo del count è [S,S+U]. Cardinality confronta questo inviluppo; Some è MATCH se S≥1, NO_MATCH se S+U=0, UNKNOWN altrimenti; None ha gli esiti opposti per gli stessi casi. All richiede il minimo N: D<N o F≥1 danno NO_MATCH; con guardia soddisfatta e U=0 dà MATCH, altrimenti UNKNOWN. Share conserva D e le proprie guardie e usa [S/D,(S+U)/D]. Si applicano 06 §§12.6 e 19.4, senza restringere l’inviluppo mediante dipendenze globali fra candidati distinti. Una base incerta conserva le proprie regole di evidenza, non diventa determinata per applicare questa formula.

L’inviluppo può includere conteggi non globalmente realizzabili; non assume indipendenza reale dei candidati. Due ordini certi O1/O2, un solo Refund con orderId ignoto ma univalore e body “nessun rimborso di quest’ordine” producono UNKNOWN per entrambi e Some UNKNOWN. Una prova globale di almeno un ordine senza rimborso sarebbe logicamente possibile, ma è esclusa dalla quantificazione approvata, non impossibile in assoluto (06 §28.24). L’executor non può scegliere una semantica più forte. La scelta N2 è stata approvata esplicitamente nella versione 1.2.

Restano obbligatorie le dipendenze interne: E considera le selezioni compatibili; P mantiene una chiave comune all’intero blocco; share mantiene part/whole, base e pesi comuni; guardie e ownership sono preservate. La composizione K non autorizza a spezzarle né a propagare prove globali fra candidati distinti.

### 9.2 Union K

```ml
type IdentitySet =
  | ObservedValues of ObservedValueCondition
  | CorrelatedEventObjects of CorrelatedEventObjectCondition
  | Opportunities of OpportunityCondition
  | AuthoritativeObjects of AuthoritativeObjectCondition
```

### 9.3 Valori osservati

```ml
type ObservedValueCondition = exists O, R, D. {
  candidates : ObservedValueBase<R, D>;
  verify : ObservedValueBody<O, R, D>;
  quantify : CandidateQuantifier<ObservedValue<R, D>>
}

type ObservedValueBase<R, D> = {
  seed : FreeBehaviorScope<ObservedCandidateUse>;
  dimension : DimensionRef<SchemaOf<seed>, GrainOf<seed>, D>
}

type ObservedValueLeaf<O, R, D> =
  | CandidateValuePredicate of CandidateDimensionPredicate<D>
  | SameRepertoireMeasureTest of ObservedLocalValueTest<O, R, D>
  | PriorHistoryPresence of {
      history : RelevantHistoryBeforeSeedPeriod<O, R, D>;
      requirement : SeenBefore | NeverSeenBefore
    }
```

Ogni `ObservedValueScope` eredita il repertorio R e la dimensione D del seme e restringe le occurrence al medesimo valore candidato. Può scegliere periodo, predicati e una proiezione di grana dichiarata compatibile; non può sostituire R con un altro repertorio raw.

`ObservedLocalValueTest` usa proprietà ammesse, riduzioni atomiche e terminali quantitativi sulle foglie di questa firma. Una dimensione di count distinct, come order o giorno, è una riduzione terminalmente consumata, non un secondo insieme navigabile. Non introduce un successivo grouping per una seconda dimensione, un winner esportabile o una nuova key.

`PriorHistoryPresence` è la forma chiusa della novità: considera tutte le occurrence pertinenti registrate e interrogabili prima dell’inizio del periodo recente del seme. Non richiede l’inizio documentato della relazione reale. La finestra precedente di uguale durata non è un sostituto. Un prefisso senza occurrence registrate è vuoto; category/identity o tempo ignoti in eventi esistenti possono rendere la risposta incerta. Retention limita la novità al patrimonio conservato. Il periodo del seme deve avere un inizio interpretabile; il requisito costa una foglia e non pubblica il bordo come nuova ancora generale.

Le forme principali sono:

```ml
(* Ripetizione: candidato prodotto, misura = ordini distinti per quel prodotto. *)
ObservedValues(base=PurchaseProducts(W),
  verify=CountDistinct(SameValueScope(W), OrderDimension) >= 3,
  quantify=Some)

(* Novità di categoria. *)
ObservedValues(base=UsageCategories(W),
  verify=PriorHistoryPresence(NeverSeenBefore, BeforeStartOfSeed(W)),
  quantify=Some)

(* Ogni contenuto recente era già noto; il non-vuoto resta in All. *)
ObservedValues(base=ViewedContent(W),
  verify=PriorHistoryPresence(SeenBefore, BeforeStartOfSeed(W)),
  quantify=All(minimumCandidates=1))
```

Sono abbreviazioni tipizzate dei costruttori dichiarati, non nuove primitive.

### 9.4 Oggetti correlati negli eventi

```ml
type CorrelatedEventObjectCondition = exists O, K. {
  candidates : EventIdentityCandidateBase<K>;
  primaryAccess : PrimaryOccurrenceAccess<K>;
  verify : CorrelatedObjectBody<O, K>;
  quantify : CandidateQuantifier<EventObject<K>>
}

type EventIdentityCandidateBase<K> = {
  primary : FreeBehaviorScope<CorrelatedCandidateUse>;
  keyRole : IdentityRoleRef<K>;
  access : IdentityRoleAccess<SchemaOf<primary>, GrainOf<primary>, K>
}

type PrimaryOccurrenceAccess<K> =
  | IdentityOnly
  | UniquePrimary of CertifiedUniquePrimaryOccurrence<K>
  | FirstPrimaryPerIdentity
  | LastPrimaryPerIdentity

type CorrelatedObjectLeaf<O, K> =
  | PrimaryPropertyPredicate of AuthorizedPrimaryPropertyPredicate<O>
  | PivotMomentPredicate of AuthorizedPivotMomentTest<O>
  | SameIdentityValueTest of CorrelatedLocalValueTest<O, K>
```

Le proprietà del seme non vengono lette da una occurrence arbitraria quando più occurrence introducono la stessa identity. `UniquePrimary` richiede una corrispondenza certificata univoca alla grana; in alternativa si dichiara il pivot first/last della sorgente primaria. `IdentityOnly` non espone proprietà di un “primo record disponibile”.

Tutte le foglie comportamentali del body sono `SameIdentityScope<O,K>`. Il periodo può essere indipendente ed esplicito, oppure ancorato al pivot quando presente. Non esiste uno scope del body che cambi key, legga la key di un altro owner o diventi globale per accidente.

Questa forma copre anche il totale pagato dell'ordine confrontato con la somma dei suoi rimborsi: `Direct(Sum(same-order Refund), >=, Local(primary.paid))`. Non richiede un oggetto Order autorevole, un denominatore positivo o un minimo monetario non richiesto.

### 9.5 Opportunità

```ml
type OpportunityOrigin =
  | FirstMessageDelivery of ApprovedMessageOpportunityRef
  | FirstCampaignDeliveryToProfile of ApprovedCampaignOpportunityRef
  | InitialTicketOpening of ApprovedTicketOpportunityRef
  | UniqueDomainStart of ApprovedUniqueStartOpportunityRef

type OpportunityBase = {
  origin : OpportunityOrigin;
  startPeriod : ObservationPeriod;
  startQualifiers : OccurrenceWhere<OriginOccurrenceSchema>;
  maturityRestriction : AllOpportunities | OnlyMatureFor of PositiveDuration
}

type OpportunitySelection = AllInBase | LatestN of Position1To10

type OpportunityCondition = exists O. {
  base : OpportunityBase;
  select : OpportunitySelection;
  verify : OpportunityBody<O>;
  quantify : CandidateQuantifier<OpportunityIdentity<base.origin>>
}

type OpportunityLeaf<O> =
  | OpportunityPropertyPredicate of OpportunityScalarPredicate<O>
  | Outcome of {
      behavior : ApprovedOpportunityOutcomePopulation<O>;
      horizon : WithinAfterStart of PositiveDuration | AfterStartUntilT;
      requirement : Occurred | DidNotOccur
    }
```

La procedura canonica è parte della forma, non una pipeline configurabile:

```text
Identifica identity e primo inizio storico pertinente
→ applica periodo/qualificatori dell'opportunità
→ applica l'eventuale selezione latest N per inizio
→ valuta il body
→ quantifica le opportunità.
```

Una riconsegna non crea una nuova opportunità della stessa identity. Per FirstMessageDelivery e FirstCampaignDeliveryToProfile l’inizio è la prima consegna registrata identificabile nell’intera storia interrogabile della chiave; soltanto dopo si applica startPeriod. Non si ipotizzano inizi esterni non registrati, ma una consegna precedente presente non può essere ignorata. Se retention la elimina, può cambiare il primo inizio registrato. InitialTicketOpening e origini certificate conservano le garanzie di lifecycle/unicità del proprio contratto. L’outcome è attribuito alla stessa opportunità oltre a essere effettivamente successivo all’inizio; le incertezze di dati esistenti non si eliminano prima del body.

Il body non offre `FirstFollowing`, altri K, scope di oggetti autorevoli o generiche misure su repertori indipendenti. Più requisiti di outcome, se ammessi nel body, restano foglie attribuite alla medesima opportunità e consumano il budget massimo quattro.

`OnlyMatureFor Δ` è una scelta esplicita della base, valutata rispetto alla deadline inclusiva start+Δ; non è aggiunta dal motore per eliminare outcome UNKNOWN. Il relativo Δ e gli orizzonti del body rimangono distinguibili e non vengono identificati senza una scelta esplicita. Latest N seleziona dopo tale restrizione della base, prima degli esiti.

### 9.6 Share dei candidati

Per dieci ordini e due operazioni Refund sul medesimo ordine:

```text
Ratio: Count(Refund operations) / Count(Purchase orders) = 2/10.
K Share: Count(orders whose body says Refund exists) / Count(base orders) = 1/10.
```

Nella seconda forma il numeratore deriva dal body sui candidati della stessa base. Non si fornisce un `numeratorScope`, non si deduplicano arbitrariamente i Refund a posteriori e non si sostituisce la base con i soli ordini dall'esito noto.

**Riferimenti normativi (specifica 06 v1.2):** §§7.2, 10.4–10.5, 12.

---

## 10. Authoritative Object Contracts: riferimenti e shape separate

### 10.1 Contract-side model minimo

Le definizioni di profile property e raw behavior contengono schema/tipo, significato corrente o event-captured, normalizzazione e requisiti informativi pertinenti; un business fact derived referenzia la definizione certificata. Un raw schema non richiede un’ontology business. Date personali e timezone mantengono i permessi specifici. Le definizioni behavior dichiarano repertorio, subject attribution, grain/projection, occurrence identity/dedup e action timestamp, con i requisiti necessari a interpretare gli eventi esistenti. Non richiedono certificati di completezza esterna, inizio della relazione reale o watermark di ingestione per i normali scope event-based. Le ulteriori garanzie di inventari, lifecycle, named scopes e CDE restano contrattuali.

Questi tipi descrivono definizioni di registry, **non nodi del Filter AST**:

```ml
type ObjectContractCore<C> = {
  identity : SemanticIdentity<AuthoritativeObjectContract>;
  objectIdentity : LifecycleIdentityDefinition<C>;
  subjectMembership : CurrentProfileMembershipContract<C>;
  inventory : InventoryCompletenessContract<C>;
  authority : AuthorityAndReconciliationContract<C>;
  validity : CurrentStateValidityContract<C>;
  currentProperties : FiniteDeclaredProperties<C>;
  lifecycleFields : FiniteAuthorizedLifecycleFields<C>
}

type AuthoritativeContractDefinition =
  | StateOnlyContract of ObjectContractCore<C>
  | NamedEvidenceContract of {
      core : ObjectContractCore<C>;
      evidence : FiniteNamedEvidenceDefinitions<C>
    }
  | CartContract of {
      core : ObjectContractCore<Cart>;
      products : CartProductsMemberDefinition
    }
  | DesignatedTrialContract of {
      core : ObjectContractCore<Trial>;
      designation : CurrentTrialDesignationContract;
      usage : TrialAttributedUsageDefinition;
      upgrade : TrialUpgradeTemporalSlotDefinition
    }
```

È una union di shape chiuse. `CartContract` non eredita tutti gli slot di `NamedEvidenceContract`; `NamedEvidenceContract` non eredita gli slot temporali del Trial. Una nuova famiglia approvata che usa una shape esistente può essere un'estensione di registry; un nuovo tipo di slot richiede invece un'estensione della union e della sua semantica.

Baseline: Subscription usa StateOnly; Ticket usa NamedEvidence con Human Response; Cart usa CartContract; Trial usa DesignatedTrialContract. Un Order con Refund è consumabile nella shape NamedEvidence solo quando esiste il contratto approvato pertinente: il MAY della specifica 06 v1.2 non viene trasformato in un inventario Order obbligatorio.

### 10.2 Le tre forme da non collassare

```ml
type NamedEvidenceScopeDefinition<C, Slot> = {
  identity : SemanticIdentity<NamedEvidenceScope>;
  ownerContract : ObjectContractRef<C>;
  meaning : EvidenceMeaningRef;
  grain : GrainRef;
  properties : FiniteTypedProperties;
  time : ActionTimeAndOrderingContract;
  association : CertifiedSameObjectMembership<C>;
  horizons : FiniteAuthorizedHorizonDefinitions<C, Slot>;
  operations : ClosedLeafOperationCapabilities;
  observation : ObservationContract
}

type CartProductsMemberDefinition = {
  owner : CartContractRef;
  membership : CurrentContainedProductsContract;
  productIdentity : IdentityRoleRef<Product>;
  memberFacts : FiniteContainedMemberFacts;
  behaviorAccess : ProfileBehaviorOnSameProductCapability
}

type TrialUpgradeTemporalSlotDefinition = {
  owner : TrialContractRef;
  behavior : DeclaredProductUpgradeRepertoireRef;
  anchor : TrialInitialStartRole;
  horizon : SinceStartUntilT;
  operations : ClosedLeafOperationCapabilities
}
```

Il subject è il profilo cui l'evidenza è attribuita; non richiede che l'attore materiale di una risposta umana sia il cliente stesso. Questo non autorizza comportamento di altri profili fuori dai ruoli di evidenza specificamente ammessi.

Un named scope è automaticamente object-correlated. Il member slot legge prodotti realmente contenuti nel Cart a T. Il temporal slot Trial Upgrade osserva invece il comportamento del profilo sul repertorio di prodotto dichiarato, senza richiedere trial ID agli Upgrade.

`ClosedLeafOperationCapabilities` è un insieme finito di tag già definiti dal linguaggio: presence/absence, count e, quando autorizzate, le riduzioni/predicate foglia elencati. Non contiene codice, `AnyRule`, nomi di operatori inventati o definizioni di join. Un capability tag non crea un costruttore nuovo: seleziona un sottotipo del catalogo esistente.

### 10.3 AST dei candidati autorevoli

```ml
type AuthoritativeObjectCondition =
  | StateOnlyObjects of exists O, C. {
      contract : ObjectContractRef<C, StateOnlyShape>;
      candidateFilter : CurrentObjectBasePredicate<C>;
      verify : StateOnlyBody<O, C>;
      quantify : CandidateQuantifier<CurrentObjectIdentity<C>>
    }
  | NamedEvidenceObjects of exists O, C. {
      contract : ObjectContractRef<C, NamedEvidenceShape>;
      candidateFilter : CurrentObjectBasePredicate<C>;
      verify : NamedEvidenceObjectBody<O, C>;
      quantify : CandidateQuantifier<CurrentObjectIdentity<C>>
    }
  | CurrentCarts of exists O. {
      contract : CartContractRef;
      candidateFilter : CurrentObjectBasePredicate<Cart>;
      verify : CartBody<O>;
      quantify : CandidateQuantifier<CartLifecycleIdentity>
    }
  | CurrentDesignatedTrial of exists O. {
      contract : TrialContractRef;
      productContext : ApprovedTrialContextRef;
      verify : TrialBody<O>;
      quantify : CandidateQuantifier<DesignatedTrialIdentity>
    }
```

`CurrentObjectBasePredicate` contiene solo predicati correnti della base candidata, mai test sulle risposte o comportamenti del body. Per un quantificatore sui ticket, selezionare prima “ticket senza risposta” non è un modo equivalente di rappresentare “tutti i ticket hanno zero risposte”.

### 10.4 Body specifici

```ml
type StateOnlyLeaf<O, C> =
  | CurrentObjectPredicate of AuthorizedCurrentObjectPredicate<O, C>

type NamedEvidenceObjectLeaf<O, C> =
  | CurrentObjectPredicate of AuthorizedCurrentObjectPredicate<O, C>
  | LifecyclePredicate of AuthorizedLifecycleScalarTest<O, C>
  | NamedEvidenceValueTest of AuthorizedNamedEvidenceValueTest<O, C>

type CartLeaf<O> =
  | CurrentCartPredicate of AuthorizedCurrentCartPredicate<O>
  | CartLifecyclePredicate of AuthorizedCartLifecycleTest<O>
  | HasCurrentProduct of exists M. {
      membership : CartProductsMemberUseRef<O>;
      verify : CartProductBody<O, M>
    }

type CartProductLeaf<O, M> =
  | ContainedMemberPredicate of AuthorizedContainedMemberPredicate<O, M>
  | SameProductValueTest of CartProductLocalValueTest<O, M>

type TrialLeaf<O> =
  | CurrentTrialPredicate of AuthorizedCurrentTrialPredicate<O>
  | TrialLifecyclePredicate of AuthorizedTrialLifecycleTest<O>
  | AttributedUsageTest of TrialUsageValueTest<O>
  | UpgradeSinceStartTest of TrialUpgradeValueTest<O>
```

Ogni body è un `ClosedBody` sulla propria leaf union. `CartBody` può contenere al massimo un member frame autorizzato; `CartProductBody` non contiene un member constructor. `NamedEvidenceValueTest` lega ogni reference al contratto C dell'owner; un `HumanResponse<Ticket@v>` non è accettato dove serve `Refund<Order@w>`.

I normali confronti V fra una riduzione named autorizzata e una proprietà locale autorizzata sono possibili, per esempio Refund sum ≥ paid total di un Order che espone entrambi. Non si rende per questo disponibile una sorgente behavior raw arbitraria nel body.

### 10.5 Date native e fatti coerenti

`LifecycleAge(initialOpenedAt)` è una lettura temporale nativa autorizzata del ruolo Ticket, semanticamente T−openedAt. È un helper chiuso del relativo test temporale, non un costruttore `SubtractTimestamp` generale. L'età dalla riapertura è un ruolo differente e non sostituisce l'apertura iniziale.

Per Cart valgono vero ultimo aggiornamento, completamento del medesimo lifecycle e contenuto corrente coerente. Per Trial la designazione corrente è certificata; `Last(TrialStarted)` non è un'alternativa al contratto. Usage richiede identity o assegnazione univoca certificata; Upgrade è il distinto slot temporale. Il body di S004 costa quattro scope: Import, Dashboard, Share, Upgrade.

I fatti del ciclo di billing corrente possono essere proprietà/date personali autorevoli, senza un traversal Subscription → Cycle. La loro coerenza e validità stanno nel contratto referenziato.

**Riferimenti normativi (specifica 06 v1.2):** §§7.3, 15, 22.

---

## 11. Certified Derived Evidence: forme e consumer separati

### 11.1 Minimo sufficiente

```ml
type CertifiedEvidenceDefinition =
  | CertifiedScalarDefinition of ScalarEvidenceSignature
  | CertifiedOccurrenceDefinition of OccurrenceEvidenceSignature
  | CertifiedMilestoneDefinition of MilestoneEvidenceSignature
  | CertifiedCurrentStateDefinition of CurrentStateEvidenceSignature
  | CertifiedReturnEpisodeDefinition of CanonicalReturnEpisodeSignature
```

La quinta forma non è un generico `TemporalEpisode<any>`: rappresenta soltanto l'episodio di ritorno canonico ammesso, quando pubblicato esternamente con un contratto equivalente. Gli altri pattern T non ricevono un insieme di episodi arbitrari. Se l'installazione non pubblica episodi di ritorno, T li riconosce nella propria forma nativa; non è introdotta una dipendenza obbligatoria dal producer.

Una proprietà storica certificata è un field della occurrence/milestone con una qualificazione semantica storica; non richiede una nuova forma di dato interrogabile mediante lookup.

### 11.2 Firma minima della definizione

```ml
type CertifiedEvidenceCommon = {
  identity : SemanticIdentity<CertifiedEvidenceDefinition>;
  meaning : EvidenceMeaningRef;
  subject : CurrentProfileSubjectContract;
  validity : EvidenceValidityContract;
  observation : ObservationContract;
  parameterSignature : FiniteClosedEvidenceParameters;
  authorizedUses : FiniteConsumerCapabilities
}

type ScalarEvidenceSignature = {
  common : CertifiedEvidenceCommon;
  valueType : ScalarType;
  semanticDomain : SemanticDomainRef;
  declaredPopulationAndPeriod : EvidencePopulationPeriodContract;
  quantitativeGuards : DeclaredEvidenceGuardContract
}

type OccurrenceEvidenceSignature = {
  common : CertifiedEvidenceCommon;
  occurrenceIdentity : OccurrenceIdentityContract;
  grain : GrainRef;
  actionTime : ActionTimestampContract;
  ordering : OrderingContract;
  properties : FiniteTypedCertifiedProperties;
  identityRoles : FiniteIdentityRoleAccesses
}

type MilestoneEvidenceSignature = {
  occurrence : OccurrenceEvidenceSignature;
  attainmentMeaning : MilestoneAttainmentContract;
  firstAttainmentGuarantees : DeclaredFirstAttainmentCapabilities
}

type CurrentStateEvidenceSignature = {
  common : CertifiedEvidenceCommon;
  authority : StateAuthorityContract;
  stateSchema : CurrentStateSchema;
  consumption : PublishedCurrentPropertySlots | ApprovedObjectContractBacking
}

type CanonicalReturnEpisodeSignature = {
  common : CertifiedEvidenceCommon;
  repertoire : RepertoireRef;
  predecessor : CertifiedPredecessorOccurrenceRole;
  returnOccurrence : CertifiedReturnOccurrenceRole;
  noInterveningActivity : CertifiedAbsenceContract;
  gap : PositiveElapsedGapContract;
  episodeIdentity : ReturnOccurrenceIdentityContract;
  authorizedUses : Resumed | RepeatedReturns | LastQualifiedReturn
}
```

`authorizedUses` nelle firme è un insieme finito di consumer tag, non una scelta di output da collegare a qualunque input. Identità oggetto e proprietà storiche sono presenti quando pertinenti alla forma e all'uso.

### 11.3 Reference e parametri, non algoritmo o chiamata composizionale

```ml
opaque type CertifiedScalarUseRef<T, ScalarSlot>
opaque type CertifiedOccurrenceUseRef<S, R, OccurrenceConsumer>
opaque type CertifiedMilestoneUseRef<S, R, MilestoneConsumer>
opaque type CertifiedCurrentStateUseRef<StateSlot>
opaque type CertifiedReturnUseRef<ReturnConsumer>

type CertifiedEvidenceUse<D, Form, Consumer> = {
  definition : CertifiedDefinitionRef<D, Form>;
  parameters : ClosedParameterBinding<D>;
  permittedUse : Proof<ConsumerAuthorized<D, Form, Consumer>>
}
```

`ClosedParameterBinding<D>` è il prodotto finito dei parametri dichiarati dalla definizione, per esempio periodo, soglia letterale e content identity/version. I valori sono letterali, riferimenti semantici ammessi e time specifications chiuse. Non sono `ReadableAtom`, output di rule, altri producer invocation o algoritmi inline.

Questo riferimento identifica la domanda di evidenza certificata da consumare, non configura come produrla. Il documento non prescrive che il producer venga chiamato durante l'esecuzione del filtro.

### 11.4 Confini di consumo

| Forma pubblicata | Uso ammesso | Uso impossibile/non autorizzato |
|---|---|---|
| Scalar | V nello slot corrente/locale autorizzato | Non compare nell'union OccurrenceSource; il timestamp metadata non lo promuove |
| Occurrence | Behavior Scope, E o P quando consumer e ruoli sono certificati | Non crea automaticamente stato corrente autorevole |
| Milestone | Stessi usi nei consumer autorizzati, preservando attainment semantics | Non equivale a un generico evento Completed |
| Current state | Proprietà correnti pubblicate oppure K attraverso l'oggetto approvato | Nessuna storia o traversal generico |
| Historical property | Predicato sulla specifica evidenza che la porta | Nessun `lookupAsOf(profileField, localTime)` |
| Canonical return episode | I soli slot di ritorno T sopra dichiarati | Non alimenta recognizer arbitrari né concatena pattern |

Una milestone passata non scade solo perché è passata: vale il proprio contratto storico. Uno scalar dichiarato corrente ma non valido a T può rendere UNKNOWN il test pertinente. Un errore tecnico del producer o della lettura non viene trasformato in UNKNOWN semantico.

### 11.5 Video S012

La definizione concreta deve certificare almeno content identity/version, durata pertinente positiva, finestra W, soglia q, copertura unica riferita a W, primo raggiungimento del 90% cumulato dall'inizio di W, identity episodio/serie e tempo pertinente. Sulla seconda milestone deve essere pubblicata la disponibilità storica dell'episodio successivo per il contesto richiesto.

Nell'AST compaiono solo un `CertifiedMilestoneUseRef`, i suoi parametri finiti e field normali della milestone. Non compaiono segment ranges, unione di porzioni video, funzioni storiche o un nodo `VideoPath`.

**Riferimenti normativi (specifica 06 v1.2):** §§7.4, 16, 18, 23.

---

## 12. P — Percorso positivo finito e un solo blocco correlato

### 12.1 Passi e selezione

```ml
type PathStep = {
  base : FreeBehaviorScope<PathStepUse>;
  select : AnyOccurrence | FirstInBase | LastInBase
}

type AdjacentDelay = Optional<ClosedDurationConstraint>
type TotalDelay = Optional<ClosedDurationConstraint>

opaque type PlainChain<N>             (* N passi; N-1 archi adiacenti *)
opaque type KeyedChain<K, N>           (* N passi; una sola key K del blocco *)
```

Ogni passo ha il proprio periodo. “Tutti i passi in W” si materializza impostando W nelle basi dei passi; non esiste una restrizione globale aggiunta silenziosamente. Questo preserva I071, dove l'errore può precedere il bordo del periodo del contatto.

First/Last di un passo viene applicato alla base dichiarata **del profilo**, prima della ricerca della tupla e prima di imporre l'uguaglianza della key del blocco. `KeyedChain` richiede che i passi espongano il ruolo di key, ma non trasforma le loro basi in `SameIdentityScope`. First/Last per identità appartiene al pivot K, non a P.

### 12.2 Struttura della correlazione

```ml
type PathLayout =
  | Independent of PlainChain<N>                         (* 2 <= N <= 5 *)
  | Correlated of exists K. CorrelatedLayout<K>

type CorrelatedLayout<K> =
  | Whole of KeyedChain<K, N>                            (* 2 <= N <= 5 *)
  | Starts of {
      block : KeyedChain<K, B>;
      bridge : PlainAdjacentEdge;
      suffix : PlainChain<S>
    }                                                   (* B>=2, S>=1, B+S<=5 *)
  | Ends of {
      prefix : PlainChain<P>;
      bridge : PlainAdjacentEdge;
      block : KeyedChain<K, B>
    }                                                   (* P>=1, B>=2, P+B<=5 *)
  | Middle of {
      prefix : PlainChain<P>;
      enter : PlainAdjacentEdge;
      block : KeyedChain<K, B>;
      leave : PlainAdjacentEdge;
      suffix : PlainChain<S>
    }                                                   (* P,S>=1, B>=2, P+B+S<=5 *)
```

La somma massima cinque è una relazione finita fra lunghezze di tuple. Può essere implementata mediante poche varianti di tuple o costruttori di catene limitate; non richiede una query ricorsiva o un indice calcolato dall'utente.

C'è un solo `exists K` e un solo `KeyedChain` per percorso correlato. `KeyedChain` non contiene `PathLayout` o un secondo blocco. La correlazione globale è `Whole`, non un flag aggiuntivo applicabile a un percorso già correlato.

Il `keyRole : IdentityRoleRef<K>` è dichiarato una volta dal blocco. Gli accessi alle proprietà di identity dei diversi source sono verificati contro quel ruolo; non sono predicati di join configurabili. Per key multivalore serve un unico valore comune all'intero blocco, non intersezioni pairwise. Il valore scelto rimane il medesimo per tutte le foglie keyed della stessa prova.

### 12.3 Slot negativi e body del percorso

Per i comuni vincoli adiacenti e per la durata totale gli estremi sono impliciti nella struttura. Per disporre in un body AND/OR limitato le assenze autorizzate, il proprietario può usare **handle di slot strutturali**, non ID di occurrence:

```ml
opaque type AdjacentIntervalSlot<O>          (* uno degli N-1 archi del layout O *)
opaque type FirstLastIntervalSlot<O>
opaque type CorrelatedInternalIntervalSlot<O, K>
opaque type WholeCorrelatedIntervalSlot<O, K>  (* esiste solo per Whole *)

type AnyBetweenSlot<O> =
  | Adjacent of AdjacentIntervalSlot<O>
  | FirstLast of FirstLastIntervalSlot<O>

type KeyReadableBetweenSlot<O, K> =
  | InternalBlockAdjacent of CorrelatedInternalIntervalSlot<O, K>
  | WholeBlockFirstLast of WholeCorrelatedIntervalSlot<O, K>

type PathNegativeLeaf<O> =
  | NoProfileBehaviorBetween of {
      slot : AnyBetweenSlot<O>;
      behavior : BehaviorPopulation<PathNegativeUse, S, R, G>
    }
  | NoSameKeyBehaviorBetween of {
      slot : KeyReadableBetweenSlot<O, K>;
      behavior : KeyReadablePopulation<PathNegativeUse, K>
    }
```

Gli handle sono dichiarati dal layout proprietario e hanno un namespace locale; le posizioni adiacenti sono un insieme finito derivato dalla catena. Non vengono introdotti `StepId` scelti dall'utente, endpoint arbitrari, reference a una sibling rule o possibilità di usare l'handle come timestamp scalare. Nella canonical equality valgono le posizioni strutturali, non l'ID temporaneo del builder.

`NoSameKeyBehaviorBetween` non accetta `AnyBetweenSlot`: l'AST non può associare una key a un arco che attraversa il confine del blocco. Per il vincolo primo–ultimo, entrambi gli estremi sono nel blocco solo nel caso Whole.

```ml
type FinalOutcomeWithoutKey<O> = Optional<ProfileOutcomeFromFinal<O>>
type FinalOutcomeWithKey<O, K> = Optional<
    ProfileOutcomeFromFinal<O> | SameKeyOutcomeFromFinal<O, K>>

type OrderedPath = exists O. {
  layout : PathLayout<O>;
  adjacentDurations : DurationsAtAdjacentEdges<O>;
  totalDuration : TotalDelay;
  negativeChecks : NoNegativeRequirements
                | BudgetedCanonicalBool2<PathNegativeLeaf<O>>;
  finalOutcome : FinalOutcomeAllowedByLayout<O>
}
```

`FinalOutcomeAllowedByLayout` è la seguente famiglia chiusa:

| Layout | Slot finale |
|---|---|
| Independent | WithoutKey |
| Whole | WithKey |
| Starts, con suffix non vuoto | WithoutKey |
| Ends | WithKey |
| Middle | WithoutKey |

Un esito finale contiene solamente presenza/assenza di comportamento dopo il finale, `Within Δ` oppure `UntilT`, più il where autorizzato. Non apre un altro percorso. Il budget complessivo conta tutte le foglie negative e l'eventuale esito, al massimo quattro. Gli estremi di intervallo restano fissi per ciascuna foglia anche sotto OR.

### 12.4 Adiacenza e ordine

“La prima azione successiva del repertorio R” usa l’assenza di azioni R registrate strettamente fra i due passi, oltre alla precedenza. R deve essere dichiarato e interpretabile; non si ipotizzano azioni esterne mai registrate e non lo si sostituisce con il source del secondo passo se la domanda richiede un repertorio più ampio. Eventi esistenti con tempo/qualificazione/identity incerti possono rendere l’adiacenza o un’assenza locale UNKNOWN quando decisivi. Gli outcome entro Δ conservano la maturazione; i contratti specializzati conservano le garanzie richieste.

La medesima occurrence non può soddisfare due passi. Timestamp uguali richiedono ordine ulteriore affidabile; il ritardo può essere zero se tale ordine esiste. I vincoli di durata sono confronti chiusi su durate adiacenti o totale primo–ultimo: non leggono proprietà arbitrarie di altri passi.

`AnyOccurrence` richiede l'esistenza di **una qualsiasi tupla interamente valida**. Non prescrive una ricerca greedy. First/Last espliciti vincolano invece le occurrence prima della ricerca e non sono sostituiti per recuperare il percorso.

### 12.5 Perché il blocco è preferito a start/end index

Un modello `steps + correlation.start + correlation.end` potrebbe essere validato correttamente, ma renderebbe naturali intervalli vuoti, invertiti e blocchi aggiuntivi in una lista di correlazioni. Prefix + Block + Suffix rende contiguità e unicità proprietà della forma.

Si conservano soltanto handle locali di **slot autorizzati** per i check che devono riferirsi a un intervallo. Non sono usati per definire la correlazione, e non sono un meccanismo generale di accesso ai passi.

**Riferimenti normativi (specifica 06 v1.2):** §§7, 13, 17, 28.13.

---

## 13. T — Catalogo temporale chiuso

### 13.1 Union canonica e forme normalizzate

```ml
type ActivityPattern =
  | DistinctActiveDays of {
      scope : FreeBehaviorScope<ActivityUse>;
      test : CardinalityTest
    }
  | CalendarDayBurst of {
      scope : FreeBehaviorScope<ActivityUse>;
      minimumOccurrencesInOneDay : PositiveInt
    }
  | RollingWindowBurst of {
      scope : FreeBehaviorScope<ActivityUse>;
      width : PositiveDuration;
      minimumOccurrences : PositiveInt
    }
  | Streak of {
      scope : FreeBehaviorScope<ActivityUse>;
      minimumConsecutiveDays : PositiveInt
    }
  | MaximumGap of {
      scope : FreeBehaviorScope<ActivityUse>;
      maximumElapsedGap : PositiveDuration
    }
  | ResumedAfterInactivity of {
      returns : CanonicalReturnDefinition;
      returnScope : ReturnObservationScope
    }
  | RepeatedReturns of {
      returns : CanonicalReturnDefinition;
      returnScope : ReturnObservationScope;
      test : CardinalityTest
    }
  | LastQualifiedReturn of exists O. {
      returns : CanonicalReturnDefinition;
      selectionPeriod : ObservationPeriod;
      verify : SingleExperienceBody<QualifiedReturnOwner<O>, ReturnOccurrenceSchema>
    }
  | QualifiedPeriodCardinality of QualifiedPeriodCondition
```

Tre forme chiuse del catalogo si normalizzano senza perdita a QualifiedPeriodCardinality:

```ml
ActiveEveryPeriod(population, periods)
  = QualifiedPeriodCardinality(population, periods,
        ActivePeriod, Exactly(NumberOfPeriods(periods)))

ActiveNOfMPeriods(population, periods, AtLeast(N))
  = QualifiedPeriodCardinality(population, periods, ActivePeriod, AtLeast(N))

ActiveNDaysPerPeriod(population, periods, D)
  = QualifiedPeriodCardinality(population, periods,
        AtLeastDistinctActiveDays(D), Exactly(NumberOfPeriods(periods)))
```

Queste sono forme tecniche d'ingresso nominate, non nuove primitive e non macro che perdono i parametri. Il modello canonico finale conserva separatamente periodo calendar, M, test interno e cardinalità esterna. Non serve mantenere tre tag semanticamente duplicati. La provenienza può conservare la frase “ogni settimana”.

`DistinctActiveDays` nel periodo complessivo non è normalizzato a “giorni per ogni settimana”: le distribuzioni fra settimane non sono equivalenti.

### 13.2 Qualified-period cardinality

```ml
type QualifiedPeriodCondition = {
  population : BehaviorPopulation<QualifiedPeriodUse, S, R, G>;
  periods : CompleteCalendarSeries;
  recognize : ClosedPerPeriodRecognizer;
  quantify : CardinalityTest
}

type ClosedPerPeriodRecognizer =
  | ActivePeriod
  | AtLeastDistinctActiveDays of PositiveInt
  | CalendarDayBurstAtLeast of PositiveInt
  | RollingBurstAtLeast of { width : PositiveDuration; occurrences : PositiveInt }
  | ConsecutiveActiveDaysAtLeast of PositiveInt
  | MaximumGapAtMost of PositiveDuration
```

Il periodo del recognizer è implicito: è **quel singolo bucket** della serie già dichiarata. Il recognizer non può fornire una propria population, cambiare calendario o scegliere un'altra finestra esterna. Repertorio, grana e same-occurrence predicates sono fissi per tutta la costruzione.

Non esiste un campo `body`, `formula`, `children`, `T`, `Rule` o `QualifiedPeriodCondition` dentro `ClosedPerPeriodRecognizer`. Non sono rappresentabili auto-annidamento, AND/OR di recognizer, formule fra bucket o trasformazioni del risultato.

M è positivo. I periodi sono giorno/settimana/mese completi e consecutivi, senza eliminare quelli vuoti o sconosciuti. Almeno N con N>M è invalido; analogamente si controlla la compatibilità della cardinalità dichiarata con [0,M] e fra i suoi estremi.

Il vuoto non ha un unico esito per tutti i recognizer: ActivePeriod fallisce su un periodo certamente senza attività; MaximumGapAtMost può riuscire se la durata dell'intero periodo non supera il limite. Non si aggiunge una guardia globale “almeno un'attività” a tutti i recognizer.

Con S periodi certamente qualificanti e U incerti, la cardinalità appartiene a [S,S+U]. Quel count rimane interno e non può essere un operando di V.

### 13.3 Burst, streak e max gap

RollingWindowBurst cerca una finestra di durata **trascorsa** fissa, semiaperta e interamente contenuta nel periodo esterno. CalendarDayBurst usa giorni del calendario scelto, anche quando non durano 24 ore. Eventi distanti esattamente Δ non stanno insieme nella stessa finestra rolling `[s,s+Δ)`.

Streak richiede giorni calendar consecutivi interamente nel periodo; una serie già interrotta può soddisfare il requisito esistenziale. MaximumGap include il vuoto iniziale, gli intervalli interni e il vuoto finale. Senza attività, il gap è la durata dell'intero periodo.

`MaximumGap.scope.period` MUST prevedere un estremo inferiore e superiore finiti, determinati semanticamente dalla forma temporale. `FullRelevantHistory` da solo è invalido. `AvailableHistory` è ammesso quando la definizione risolta fornisce un intervallo dichiarato con entrambi gli estremi finiti; sono ammessi anche gli altri periodi delimitati già autorizzati per quello scope. Non si inferisce un inizio dal primo evento o da una policy di retention. Un estremo strutturalmente non previsto è invalidità di configurazione; un estremo previsto ma di valore ignoto per il profilo può dare UNKNOWN quando decisivo. Restano start<end per l’assoluto e la possibilità di un ciclo corrente dinamico vuoto all’inizio, con gap=0. Il vincolo è un controllo dei parametri/definizioni esistenti, non un nuovo costruttore AST (06 §§14.4, 28.23).

Per `MaximumGapAtMost` di QPC, gli estremi finiti sono già quelli del bucket calendar completo. Burst e streak mantengono le proprie finestre/giorni finiti e la semantica esistenziale; il vincolo di MaximumGap non aggiunge loro un divieto su FullRelevantHistory. I ritorni mantengono il predecessore registrato e i propri intervalli; non si inventa un estremo storico globale.

### 13.4 Ritorni canonici

```ml
type CanonicalReturnDefinition =
  | FromActivity of {
      population : BehaviorPopulation<ReturnUse, S, R, G>;
      minimumInactivity : PositiveDuration
    }
  | FromCertifiedReturnEpisodes of CertifiedReturnUseRef<ApprovedReturnConsumer>

type ReturnObservationScope =
  | ReturnTimeIn of ObservationPeriod
  | EntirePauseIn of BoundedObservationPeriod
```

`ReturnTimeIn W` pone r in W e non restringe implicitamente il predecessore u a W. `EntirePauseIn W` richiede u, r e la pausa intera nel periodo, come I092. Non sono intervalli liberamente combinabili fra timestamp selezionati.

Un ritorno FromActivity richiede u ≺ r **entrambi registrati e interrogabili**, consecutivi nel repertorio qualificante, ritardo almeno Δ e nessuna attività registrata intermedia. Non richiede certificati esterni sulla pausa. La prima attività registrata senza predecessore non è un ritorno, anche dopo retention; il bordo di conservazione non inventa u. Eventi esistenti di tempo o qualificazione incerti restano pertinenti. Ogni r identifica un episodio; predecessori remoti non moltiplicano il count. FromCertifiedReturnEpisodes conserva invece i requisiti del contratto certificato.

`LastQualifiedReturn` seleziona l'ultimo ritorno che soddisfa la definizione di pausa nel proprio periodo di selezione, poi verifica recency e body. Espone come Self soltanto il ritorno r e i fatti autorizzati su di esso; non mette a disposizione un selettore arbitrario del predecessore o una nuova data personale globale. Il token del ritorno non è quello di una E sorella.

Per S011 il requisito minuti negli ultimi sette giorni resta una V indipendente; la mancata consegna Bentornato dopo r rimane invece nel body del ritorno.

### 13.5 Forme che non aggiungono varianti T

Aumento/calo è V con scope espliciti, unità e baseline. “Ha smesso” è una macro AND di presenza precedente e assenza recente, con gli ulteriori requisiti espliciti quando richiesti. Ultimi-due-intervalli è il preset E. Non esistono nodi opachi `Churn`, `Engagement`, `TypicalCadence` o `Median`.

**Riferimenti normativi (specifica 06 v1.2):** §§7.2, 14, 16.2, 28.9, 28.15.

---

## 14. Root filter algebra e supporto alla valutazione ternaria

### 14.1 Algebra root

```ml
type Rule =
  | V of RootTypedComparison
  | E of SelectedExperience
  | K of IdentitySet
  | P of OrderedPath
  | T of ActivityPattern

type ScopedRule = { calendar : CalendarChoice; semantic : Rule }

type Filter = {
  defaultCalendar : CalendarDefinitionRef;
  root : AllProfiles | Constrained of CanonicalBool<4, ScopedRule>
}
```

`RootTypedComparison` contiene le 26 condizioni record condivise (`RecordCondition` tramite `PropertyTest`), gli ulteriori property test governati, i cinque terminali di §7.1 e i test di distribuzione di §7.4, con `C=Root`. `LocalScalarAccess<Root,T>` è il tipo vuoto. Non ci sono local references nella top-level Rule union. Questo è il **modello canonico di Behavioral** su profilo corrente, non una sostituzione del normale `Applies` per eventi e utenti importati nei contesti Record.

### 14.2 Booleani canonici alternati

La famiglia seguente è definita per d in {0,1,2,3,4} nell’algebra root; le varianti body rimangono limitate a d≤2:

```ml
type CanonicalBool<0, L> = Leaf of L

type CanonicalBool<d, L> where d>0 =
  | Leaf of L
  | And of AndGroup<d, L>
  | Or of OrGroup<d, L>

type AndGroup<d, L> = AtLeast2<Leaf<L> | OrGroup<d-1, L>>
type OrGroup<d, L>  = AtLeast2<Leaf<L> | AndGroup<d-1, L>>

type AndGroup<0, L> = Never
type OrGroup<0, L> = Never
```

Un gruppo conta come un livello. Massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. La radice pubblica è un gruppo AND/OR; l’albero pubblico mantiene gruppi singleton, gruppi consecutivi dello stesso operatore e ordine originale; il canonical AST è distinto. La forma alternata incorpora il flattening dei gruppi dello stesso operatore. Gli atomi non sono gruppi. Un boolean group non contiene direttamente soltanto `Rule[]`, perché deve rappresentare anche il nesting AND/OR consentito; le sue foglie sono rule e i suoi figli interni sono esclusivamente gruppi booleani.

`AllProfiles` rappresenta il filtro pubblico **assente**, che non aggiunge vincoli. Non è `And([])` e non è un sesto tipo di Rule. I gruppi pubblici esplicitamente vuoti sono invalidi; i singleton sono validi e **si preservano** nel JSON pubblico/persistito, pur potendo essere eliminati soltanto nella rappresentazione canonica interna. La canonicalizzazione non cambia `core.Filter`, `state.Where` o `Where.Equal` come effetto automatico.

### 14.3 Risultati

```text
evaluateRule : Rule × CurrentProfile × T × Evidence → MATCH | NO_MATCH | UNKNOWN
membership(Filter,T) = { p nella platea Profiles : evaluateFilter(p,T) = MATCH }
```

Questa firma stabilisce il dominio semantico, non il protocollo esecutivo. Un fallimento tecnico è esterno al codominio della valutazione semantica riuscita.

| A | B | AND | OR |
|---|---|---|---|
| MATCH | MATCH | MATCH | MATCH |
| MATCH | NO_MATCH | NO_MATCH | MATCH |
| MATCH | UNKNOWN | UNKNOWN | MATCH |
| NO_MATCH | NO_MATCH | NO_MATCH | NO_MATCH |
| NO_MATCH | UNKNOWN | NO_MATCH | UNKNOWN |
| UNKNOWN | UNKNOWN | UNKNOWN | UNKNOWN |

Le righe simmetriche sono equivalenti. Le guardie usano la medesima logica. Nel percorso ternario Behavioral **non** si converte UNKNOWN a falso prima della composizione delle condizioni e dei quantificatori; il cambio MATCH → UNKNOWN rimuove il profilo dalla membership corrente. L’interfaccia Record `Applies` conserva invece `bool`: per il solo AND/OR esterno di foglie **già valutate**, il `false` è applicabilità insufficiente, non diagnostica di NO_MATCH. I risultati SQL `TRUE/FALSE/NULL` sono interpretati dalla query chiamante, senza cambiare il significato del filtro.

### 14.4 Requisiti dichiarati dai contratti

Per consentire la valutazione devono essere referenziabili, quando pertinenti:

| Requisito | Informazione nel contratto |
|---|---|
| Adeguatezza event-based | Interpretabilità delle occurrence registrate pertinenti: repertorio, grana, valori, identity, tempo e ordine; vuoto senza eventi ipotetici esterni |
| Storia | Patrimonio registrato interrogabile, eventuale intervallo esplicito e limiti noti di retention/cancellazione; nessuna certificazione dell’inizio reale per first/mai/novità |
| Ordine | Tempo dell'azione; ordine aggiuntivo affidabile, se disponibile; ambiguità non risolte |
| Identity | Namespace, equivalenza, deduplicazione alla grana, key multivalore, lifecycle identity |
| Quantità | Unità, dominio, precisione, segno/pesi non negativi e limiti effettivamente certificabili |
| Stato e completezza contrattuale | Autorità, riconciliazione, inventario, lifecycle, momento di validità, attualità e copertura specificamente richiesta da named scopes/CDE |
| Milestone | Identità e periodo, attainment e first-attainment quando richiesto, fatti storici certificati |
| Episode | Per FromActivity: predecessore registrato, ritorno, consecutività e pausa nella storia interrogabile; per episodi certificati: garanzie della definizione |

Un contratto può dichiarare una capacità di certificazione senza garantirla per ogni profilo. Un campo malformato nei dati è evidenza invalida per il test pertinente; una definizione incompatibile è invece invalidità statica.

Ogni combinazione fonte/schema/predicato ammessa nello **behavioral profile filtering** MUST avere significato osservabile e requisiti informativi definiti, invarianti rispetto all’executor. L’incertezza decisiva di dati esistenti o di garanzie contrattuali segue 06 §§19.3–19.7. La sola assenza di occurrence nel dominio registrato, anche dopo retention o senza utente corrispondente in una sorgente, non è UNKNOWN. Una capacità strutturale indispensabile assente rende la capability non disponibile, non un insieme vuoto né UNKNOWN per tutti. Non si aggiungono registry ai normali filtri Record o correzioni retroattive agli executor.

#### 14.4.1 Deroga approvata per il testo nativo

**Approvata dall’utente il 13 novembre 2026.** In deroga al requisito di invarianza rispetto all’executor di §14.4, i predicati Record ordinari su colonne native String conservano le regole di confronto del percorso SQL Record esistente. Uguaglianza, diversità, appartenenza, ordine, intervalli, contenimento, prefisso e suffisso seguono le regole della colonna fisica nel warehouse del workspace, anche nei body che ammettono tali predicati Record.

**Estensione successiva approvata dall’utente:** la deroga comprende anche `contains` e `does not contain` sugli elementi String di proprietà dichiarate `Array(String)` nello schema Krenalis, conservando il percorso SQL Record esistente. PostgreSQL usa l’uguaglianza con la collation della colonna array; Snowflake confronta letteralmente gli elementi tramite la rappresentazione JSON già adottata. Lo stesso filtro può quindi dare esiti diversi fra i warehouse o dopo una modifica della collation PostgreSQL, anche con lo stesso schema Krenalis. Il tipo dichiarato `Array(String)` resta distinto da un valore di tipo JSON che contiene un array: quest’ultimo conserva il confronto letterale e resta fuori dalla deroga.

Lo stesso filtro può dare esiti diversi su warehouse differenti o dopo una modifica delle regole di confronto della colonna, anche con lo stesso tipo Krenalis. Krenalis non rileva, registra, verifica o versiona queste regole. La deroga non introduce registry, nuovi tipi, revisioni semantiche o rilevamenti delle collation.

Un errore del warehouse dovuto alle regole di confronto resta un fallimento tecnico dell’operazione, non NO_MATCH, UNKNOWN o un risultato parziale.

Restano fuori dalla deroga i confronti letterali previsti per i selettori tecnici delle sorgenti, il confronto interno delle identità dei messaggi per deduplica e selezione, i confronti sulle stringhe JSON e le valute governate. Un normale predicato Record su una colonna String come `messageId`, `type`, `event` o `name` segue invece le regole della colonna: il nome del campo non lo trasforma in un selettore tecnico o in un confronto interno d’identità.

La deroga riguarda soltanto il confronto dei valori String: non modifica il codice degli executor, le regole di assenza/null o dell’array vuoto, gli operatori di stato, la composizione ternaria o gli altri contesti Record esistenti. Un’eventuale futura politica uniforme per il testo nativo richiederà una decisione di compatibilità separata.

### 14.5 Livelli di stato tecnico

```ml
type StaticAssessment =
  | InvalidShape of ShapeViolation
  | InvalidAgainstRegistry of DefinitionCompatibilityViolation
  | UnresolvedSemanticReference of MissingMeaningDefinition
  | ValidButUnavailable of MissingWorkspaceCapability
  | ReadyForEvaluation
```

La distinzione tra reference irrisolta e capability assente è necessaria **per le nuove costruzioni governate**. Se la definizione semantica versionata è conosciuta e valida ma manca l'adapter nel workspace, il filtro è valido ma non attivabile. Se non si conosce nemmeno la definizione referenziata, non si può attestare la validità semantica: lo stato è irrisolto, non “UNKNOWN per tutti i profili”. Le `RecordCondition` sui path esistenti seguono invece la validazione schema-aware già presente in `core.Filter`: un operatore o operando incompatibile è rifiutato come filtro invalido, **senza richiedere una `meaningRevision` o il registro governato**. Questa fase non ridefinisce i codici d’errore pubblici.

Una definizione risolta che non autorizza il consumer è `InvalidAgainstRegistry`. Un contratto Ticket valido senza adapter di workspace è `ValidButUnavailable`. Un repertorio Purchase configurato con zero eventi per il profilo è invece valutabile su un insieme vuoto. Un Ticket con inventario/stato non determinabili o con una risposta registrata di classificazione umana/automatica ignota può produrre UNKNOWN. Un errore di lettura, anche dopo alcune righe, è `ExecutionFailure`, non uno dei tre truth values.

**Riferimenti normativi (specifica 06 v1.2):** §§3, 7.5, 19–20.

---

### 14.6 Vista coerente assunta e responsabilità rinviate

La semantica assume un unico T e una vista coerente di profili, associazioni ed evidenze. IR/pubblicazione non atomica e acquisizione/attribuzione KPID sono lavori infrastrutturali rinviati; il matching valido è una precondizione fornita dal sottosistema destination, separata dalla membership. Nessuna garanzia attuale viene attestata. Le latenze tecniche non sono automaticamente UNKNOWN e non si aggiungono watermark, certificati di ingestione, tipi ReadView, generazioni o materializzazioni al linguaggio/AST. Questi lavori non bloccano il progetto del compilatore e del piano fisico, ma possono essere prerequisiti di rilascio. Errori o esecuzioni parziali non forniscono una normale valutazione semantica. Il rapporto 12 resta una proposta architetturale non approvata, da riconsiderare separatamente rispetto alla baseline normativa 1.2 (06 §3.2).

## 15. Static validation model: invarianti principali

Questa sezione specifica **quali proprietà vanno provate e a quale livello**, non il protocollo di validazione della Fase 4B.

Nella tabella: **S** = invalid shape/parametro della forma; **R** = incompatibilità con definizione risolta; **D** = definizione semantica irrisolta, validità non attestabile; **A** = AST semanticamente valido ma capability indisponibile nel workspace; **U** = possibile UNKNOWN sul profilo, salvo prova decisiva; **N** = guardia certamente falsa, quindi NO MATCH; **F** = execution failure.

“Type shape” comprende i tipi nominali a costruttore privato per valori limitati e i piccoli indici strutturali. Per un riferimento esterno la shape conserva il tipo atteso; il registry dimostra che la definizione lo offre davvero.

| Invariant | Enforced by type shape | Static registry validation | Runtime evidence | Outcome if violated |
|---|---|---|---|---|
| Esattamente V/E/K/P/T alla root Behavioral | Rule union chiusa; `RecordCondition` è foglia di V | No | No | S |
| Nessun NOT generico | Nessun costruttore | No | No | S |
| Soggetto sempre profilo corrente | Nessun subject selector locale | Contratti single-profile | Identità del profilo pertinente | S/R; dati incerti U |
| Stesso T | T solo ambientale | Nessuna definizione può sostituirlo per l'uso corrente | Vista coerente | F se valutazione coerente non eseguibile; evidenza non valida può dare U |
| Filtro privo di condizioni | AllProfiles distinto | No | Platea Profiles | Nessun errore; include platea |
| Gruppo esplicitamente vuoto | AtLeast2 canonico; input non vuoto prima della normalizzazione | No | No | S |
| Singleton group | Eliminato nella forma canonica | No | No | Non è un nodo canonico |
| Nesting root <=4 | Massimo quattro livelli di gruppi AND/OR, radice inclusa; `CanonicalBool<4, …>` interno | No | No | S |
| Massimo 100 nodi discendenti pubblici, radice esclusa | Conteggio di condizioni e gruppi **prima della canonicalizzazione**, anche se semplificabili; i budget interni delle nuove primitive restano separati | No | No | S |
| Nesting contestuale/where <=2 | Famiglia booleana limitata | No | No | S |
| Type compatibility scalare | Field e Literal indicizzati | Firma dei field | Valori conformi | S/R; record invalido U |
| Identifier domain e namespace | Tipi nominali distinti | Mapping di identità | Key individuale e valore | R; dati incerti U |
| Unità/valuta confrontabile | Tipi e witness di operazione | Unit/currency/domain definitions | Unità dinamica e valore | R; mismatch esplicito può dare N; ignoto U |
| Count ordini vs count Refund | CountMeaning separato; compatibility specifica | Grane dichiarate | Dedup affidabile | R/U, non errore per la sola differenza delle grane |
| Operatore legale per il tipo | Union/operator instance ristretta | Tipo del field | No ulteriore coercizione | S/R |
| Null distinto da missing | Costruttori distinti; null non è un literal normale | Property shape | Stato del campo nel record o garanzia del contratto | U se il dato necessario non è determinabile |
| Set property senza traversale | Operatori esistenziali chiusi | Element type | Set e valori | R/U |
| Una sola verifica owned-member nello stesso where | Indice 0/1; inner predicate senza member | Owned-list schema | Appartenenza e campi del membro | S/R/U |
| Nessun aggregate nel where | Predicate union non contiene reduction | No | No | S |
| Input di reduction sempre scope foglia | AtomicReduction non accetta atom o terminale come input | Operazione consentita dal source | Popolazione e valori | S/R/U |
| Sum additiva e uniforme | AdditiveQuantitativeField | Additività, unità, grana | Segno/valori esistenti; garanzie contrattuali quando richieste | R/U |
| Average/min/max minimo >=1 | PositiveInt obbligatorio | Operazione/field ammessi | Campione | S; guardia falsa N, incerta U |
| Sum vuota = zero, avg vuota non definita | Tag della riduzione | Unità dichiarata | Vuoto certo o possibile | Semantica del tag; non fallback |
| Count distinct una dimensione | Un solo dimension slot | Equality/domain/projection | Valori distinti incerti | R/U |
| Arithmetic non ricorsiva | TerminalTest disgiunto da ReadableAtom | No | No | S |
| Soglia aritmetica costante | Literal-only threshold | Tipo/unità | No | S/R |
| Difference può avere delta negativo | SignedDeltaLiteral separato da Count | Delta unit | Valori | R/U |
| Ratio senza unità composte arbitrarie | RatioCompatible ristretto | Compatibilità delle unità | Operandi | R/U |
| Denominatore positivo | Guardia intrinseca | Eventuali limiti quantitativi certificati | Valore o intervallo certamente positivo | N se non positivo certo; U se incerto |
| Share part/whole | Un solo whole; part solo predicate | Peso/grana/field | Qualificazione e peso | S/R/U |
| Share threshold in [0,1] | Fraction0To1 | No | No | S |
| Peso share non negativo | NonNegativeAdditiveWeight | Contratto di peso | Conformità dei pesi | R; evidenza invalida U |
| Unicità del winner | Guardia intrinseca; nessun winner operand | Dominio dimensione | Frequenze e massa ignota, verità dell’intero test | Parità certa N; U solo se l’ignoto può cambiare la verità del test, non per la sola identità incerta |
| Local reference owner-correct | Token generativo e slot-specific | Proprietà del ruolo | No lookup esterno | S/R |
| Nessuna variabile cross-condition | CloseOwner rimuove il binding | No | No | S |
| Body K specifico | Discriminated leaf unions | Signature del contratto | No ampliamento runtime | S/R |
| Massima profondità contestuale | Owner principale, eventuale solo Cart member, poi foglie | Cart member capability | No | S/R |
| Massimo quattro scope subordinati | Indice di budget per owner | Scope operations | No | S |
| Guard static consistency | Parametri controllati del medesimo test; nessun guard AST libero | Tipi e vincoli dichiarati degli stessi operandi | Non richiede inferenze fra sibling | S per minimi/bound configurati incompatibili; false guard runtime resta N |
| Guardie non aggiungono scope | Guardia posseduta dal relativo operando | No | Evidenza dello stesso scope | S se si tenta un operand arbitrario |
| Nth/latest N 1–10 | Position1To10 | Repertorio, grana e ordinamento interpretabili | Esistenza di N e selezione fra eventi registrati, senza prefissi/coda esterni certificati | S; assenza certa N; selezione incerta U |
| Base → select → verify | Campi/fasi distinti | Fonte selezionabile | Ordine e qualificazione | S/R/U; nessun fallback |
| Last/previous stesso field | Un solo field nel preset | Tipo quantitativo quando richiesto | Ultime due identificabili | S/R/U |
| Last-two-intervals ultimi tre | Preset senza indici | Ordering | Tre occurrence e due durate positive | N/U |
| FirstFollowing non ricorsivo | Payload senza scope relativo o selector annidato | Source autorizzato | Primo seguito e ritardo | S/R; inesistenza certa N; dubbio U |
| K observed stesso repertorio/dimensione | Owner-bound scope type | Proiezioni compatibili | Valori e storia | S/R/U |
| Novità prima dell’inizio di W | PriorHistoryPresence chiuso | Repertorio/dimensione e bordo W, non inizio della relazione reale | Eventi registrati prima di W; valori/identity/tempi incerti restano pertinenti | R/A se manca una capacità necessaria; U per dati esistenti decisivi |
| K event correlation una identity | KeyRole unico; scope implicitamente restricted | Ruolo comune/namespace | Key di ogni occurrence | R/A/U |
| Proprietà primarie K non arbitrarie | UniquePrimary o pivot dichiarato | Unicità primaria/ordine | Selezione e valori | R/U |
| Candidate base prima del body | Payload separati | Fonte candidata/inventario | Candidati possibili | S/R/U |
| All non vacuo | Minimo positivo intrinseco | Limiti di inventory quando certificati | Candidati e body | Vuoto certo N; incertezza U |
| Candidate share conta candidati | Quantificatore K, non ratio di occurrence | Candidate identity | S/F/U/D dei body; denominatore condiviso | S/R/U |
| K composizionale su base determinata | CandidateQuantifier applicato ai body individuali | Nessuna inferenza globale fra candidati distinti | D=S+F+U; inviluppo [S,S+U] | U conservabile anche con una prova globale più forte; non scelta dell’executor |
| Opportunità prima identificata storicamente | Origin chiusa distinta dal periodo | Start/identity contract | Primo inizio per key | R/A/U |
| Latest N opportunities per start | Selector distinto precedente al body | Ordering/start capability | N inizi identificabili | S/R/U o N se insufficienti certi |
| Immature non escluse implicitamente | MaturityRestriction esplicita nella base | Orizzonte supportato | T e dati esistenti pertinenti | U sul body quando pendente |
| Object contract shape | State/Named/Cart/Trial distinti | Family/slot signature | Stato e inventario | S/R/A/U |
| Named evidence same object | Named scope reference legato a C e O | Certified association | Appartenenza/identity | R/A/U |
| Trial Upgrade non object-correlated | Tipo temporale distinto, source fisso | Trial slot definition | Start e comportamento del profilo | S/R/A/U |
| Cart membro realmente contenuto | Member use ref del Cart owner | Current membership contract | Contenuto coerente a T | R/A/U |
| CDE forma corretta | Ref separati per scalar/occurrence/milestone/state/return | Consumer authorization | Adeguatezza del dato | S/R/A/U |
| CDE params non composizionali | Binding finito, no atom/formula | Parameter signature | Evidenza per W/q/T pertinenti | S/R/U |
| Historical property non current fallback | Semantic property role distinto | Subject/time meaning | Valore storico certificato | R/A/U |
| P 2–5 passi | Catene limitate | No | Occurrence distinte | S; tuple impossibili NO_MATCH |
| Un solo correlation block | Un solo KeyedChain nel layout | No | No | S |
| Contiguità del blocco | Prefix/block/suffix | No | No | S |
| Una sola correlation key | Un solo K nel blocco | Role access dei source | Valore comune a tutti | S/R; intersezione vuota certa NO_MATCH; dubbio U |
| First/Last P prima della correlation | PathStep.base è profile-scoped | Selettore autorizzato | Selezione/ordine | S/R/U |
| Negative key visibility | KeyReadableBetweenSlot separato | Source supporta la medesima key | Identity ed assenza | S/R/U |
| Final key visibility | Famiglia per layout | Mapping key outcome | Esito e key | S/R/U |
| Nessun accesso arbitrario fra passi | Slot endpoints chiusi, nessun field lookup su altri passi | No | No | S |
| P any tuple, non greedy | AnyOccurrence semantic tag | Ordine affidabile | Possibilità di tuple valide | NO_MATCH solo se tutte escluse; altrimenti U o MATCH |
| Durate/finestre staticamente compatibili | Tipi di bound + checked parameters | Unit/calendar | Valori dinamici | S per parametri incompatibili; U per evidenza |
| ProfileDate vs occurrence anchor | Tipi e window constructors distinti | Anchor permission | Date/ordine | S/R/U |
| Absolute start < end | Checked absolute interval | Time interpretation | No | S |
| Ciclo corrente può essere vuoto all'inizio | Costruttore SinceStartUntilT | Current-cycle contract | Start=T ammesso | Non invalido per il solo vuoto |
| Deadline inclusa e maturazione T>d | Tag WithinAfter... | Orizzonte e dati temporali interpretabili | T, eventi registrati fino a d inclusa; garanzie di contratto se richieste | Pending U; risposta certa può confutare subito |
| Calendar vs elapsed | Tipi distinti | Calendar/timezone definitions | Fuso corrente se usato | R/U |
| T catalogo chiuso | ActivityPattern union | Source/consumer | Evidenza pattern | S/R/U |
| MaximumGap con entrambi gli estremi finiti | Checked period; FullRelevantHistory da solo non ammesso | AvailableHistory e altri periodi autorizzati devono prevedere entrambi i bordi | Valori degli estremi sul profilo | S se la forma non prevede un bordo; R se la definizione risolta non lo fornisce; U se il valore previsto è ignoto e decisivo |
| QPC recognizer autorizzato | Sei varianti, nessun child Rule/T | Population/calendar | Risultati per bucket | S/R/U |
| QPC non annidabile | Nessun QPC nel recognizer | No | No | S |
| QPC serie finita completa e fissa | CompleteCalendarSeries, M positivo | Calendar | Bordi e dati esistenti interpretabili; nessun evento registrato significa bucket vuoto | S/R/U |
| QPC cardinalità compatibile con M | Checked cardinality; es. AtLeast N>M invalido | No | S/U dei periodi | S; altrimenti valutazione su [S,S+U] |
| Periodi vuoti o ignoti non eliminati | Nessun filter dei bucket | No | Test applicato a tutti | Violazione del significato, non normalizzazione ammessa |
| Ritorno con predecessore vero | CanonicalReturnDefinition chiusa | Repertorio/episode contract | u e r registrati, nessun intermedio registrato; contratto degli episodi certificati | Primo registrato senza predecessore NO_MATCH; dati decisivi incerti U |
| Un ritorno = un episodio | Identity del ritorno | Event/episode identity | Dedup | R/U |
| Semantic reference stabile | Meaning revision obbligatoria | Definizione immutabile risolta | No display dependency | D/R |
| Capability concreta disponibile | Non basta la shape | Binding di workspace | Non è un fatto del singolo profilo | A |
| Fallimento tecnico distinto da UNKNOWN | Separazione StaticAssessment / Truth / failure | No | Esito tecnico | F |
| Provenance non necessaria alla semantica | Metadata separato | Tutti i semantic refs nell'AST | No | AST incompleto se dipende dalla recipe |

La tabella non richiede un risolutore generale della soddisfacibilità dei filtri. Si controllano i vincoli statici definiti dalla forma: tipi, limiti, intervalli invalidi, capacità e incompatibilità di parametri. Un normale requisito che un profilo non soddisfa non viene trasformato in invalid configuration.

---

### 15.1 Ammissione per contesto operativo

Prima di risolvere e valutare una rule, il sistema determina se l’operazione appartiene al livello Record o Behavioral. Il validatore server deve autorizzare le nuove forme di `FilterRule` **solo** in Profiles API/Admin e nelle destination connections che esportano profili verso app, file e database. Altre pipeline mantengono soltanto le condizioni Record dove sono già ammesse; divieti come Source/Database/User restano invariati. Il rifiuto per contesto non è `NO_MATCH`, non è SQL `NULL` e non è `ValidButUnavailableInWorkspace`: è un errore di ammissibilità dell’input.

La validazione della struttura pubblica è distinta dalla validazione del corpo di E/K/P/T, dal registry e dalla disponibilità delle evidenze. Massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa; i gruppi esplicitamente vuoti restano invalidi. I test su `RecordCondition` riusano la validazione esistente, senza introdurre requisiti retroattivi di significato revisionato per raw path salvati.

**Non definito in questa fase:** nomi/tag wire delle nuove `FilterRule`, serializzazione e migrazione della persistenza, piano fisico e integrazioni Admin. Coerenza IR/acquisizione e matching valido sono assunzioni di §14.6 con lavori autonomi rinviati; nessun protocollo snapshot/generazioni, watermark o materializzazione del rapporto 12 viene approvato. Le questioni operative rinviate non bloccano la progettazione corrente.

## 16. Canonicalization policy

### 16.1 Obiettivo e limiti

La canonical equality è **uguaglianza strutturale dopo un insieme dichiarato di riscritture sicure**. Non promette di decidere l'equivalenza logica di ogni filtro possibile. Hash, caching e deduplication possono usarla senza richiedere un theorem prover. L'hash dell'AST identifica la richiesta, non da solo un risultato di membership: T, evidenza pertinente e relative revisioni rimangono rilevanti per il riuso di un risultato; non si progetta qui il relativo execution contract.

Si distingue l’ammissione nel linguaggio dalla normalizzazione: una costruzione semantica con gruppo esplicitamente vuoto, scope eccessivi, slot illegali o parametri incompatibili non diventa lecita solo perché una successiva semplificazione potrebbe cancellarne una parte. I budget vengono controllati prima delle riscritture che eliminano usi. **Il documento pubblico, il persistito `state.Where`, `Where.Equal`, i change detector e i fingerprint esistenti non sono riscritti implicitamente** dalla canonical equality interna.

### 16.2 Riscritture canoniche

| Area | Policy |
|---|---|
| AND/OR | Elimina singleton; flatten dello stesso operatore; ordina figli per chiave strutturale deterministica; elimina duplicati identici |
| Profondità | La rappresentazione canonica è alternata e limitata; UI grouping hints non aggiungono gruppi semantici |
| Letterali numerici | Forma esatta nella precisione/unità dichiarata; percentuali come frazioni esatte; nessun epsilon |
| Testo/enum/identifier | Normalizzazione soltanto se la definizione la dichiara; niente case-fold o Unicode folding universale |
| Durate | Normalizza scale trascorse equivalenti; non trasforma un giorno calendar in 24 ore |
| Timestamp | Normalizza istanti non ambigui; conserva semantica calendar e disambiguazione quando pertinente |
| Membership literal list | Dedup e ordinamento **solo nella forma canonica interna**, preservando i kind dei valori JSON e la membership mista; mai mutare automaticamente il payload pubblico né i controlli di ordine/uguaglianza già esistenti |
| Cardinalità | Nel quantificatore K: AtLeast(1) → Some, Exactly(0) → None, Between(n,n) → Exactly(n) e poi la regola precedente; preserva minimi di All e share |
| Count interval | Può normalizzare estremi interi aperti a estremi inclusivi solo con equivalenza esatta e dominio noto; non lo fa per money/date in modo implicito |
| Default | Materializza periodo, calendario effettivo, grana e minimi normativi; “nessun minimo extra” è distinto da un minimo di recipe |
| E selectors | Nth(1) → First; first/last qualifying determinati dal where della base, non da label |
| T aliases | Every/N-of-M/N-days-per-period → QualifiedPeriodCardinality con serie e recognizer equivalenti |
| Riferimenti | Risolve alias solo quando il registry ne certifica la stessa definizione immutabile; non fonde raw/business per somiglianza di nomi |
| Owner | Normalizzazione per ruolo/posizione strutturale, modulo rinomina del token locale |
| Simmetria diretta | Può ordinare operandi di Equal/Different invertendo correttamente i relativi ruoli; non commuta difference, ratio o recent/baseline |
| Proprietà vs direct | Un confronto governato semplice può essere ricondotto alla forma property corrispondente **soltanto** se equivalenza di tipi, operatori e missing/JSON è dimostrata; `RecordCondition` non viene riscritto in una forma non equivalente |

L'ordine dei figli booleani è indipendente da display name e token locali casuali. La chiave comprende tag delle union, literal normalizzati, firme semantiche dei reference, guardie e parametri. Il protocollo binario/stringa da dare in input all'hash è rimandato alla fase appropriata.

### 16.3 Non-riscritture vincolanti

Non si normalizzano fra loro:

- base-prefilter e post-selection verify; first/last e first/last qualifying;
- ratio e share; quota per peso e per occurrence; count ordini e count Refund;
- current profile, event-captured, certified historical e authoritative current state;
- correlated e independent; key globale e key limitata al blocco;
- rolling elapsed e calendar; full relevant history e available history;
- `Within Δ` negativo e `UntilT`; bordi inclusivi di deadline e finestre semiaperte;
- All e None dell'opposto; All e una equality di conteggi che richiederebbe inferenze non previste;
- ultimo valore e ultimo non-null; sum di una popolazione e sum dei soli valori conosciuti;
- stato corrente e latest lifecycle event; primo raggiungimento in W e completion lifetime;
- distinte selection base, activity window e deadline.

Non si semplifica `x=x` a true quando x può essere ignoto. Non si introduce un falso/vero costante al posto di guardie o requisiti di esistenza. Non si usa distribuzione in DNF/CNF come policy ordinaria: può esplodere la struttura e duplicare contesti senza beneficio necessario. La canonicalization non sposta una foglia fuori dal suo owner, neppure per ottenere un gruppo più piatto.

Un'esecuzione può condividere la lettura fisica di scope identici, ma ciò non elimina le posizioni d'uso semantiche né modifica i limiti di costruzione.

**Riferimenti normativi (specifica 06 v1.2):** §§3, 7, 10–14, 17–20, 29.

---

## 17. Recipe, presentation e provenance sono un documento separato

```ml
type PresentationProvenance = {
  entries : List<PresentationEntry>
}

type PresentationEntry = {
  presentationIdentity : PresentationItemId;
  semanticTargets : NonEmpty<SemanticNodeAddress>;
  recipe : Optional<RecipeIdentityAndVersion>;
  displayIntent : Optional<DisplayIntent>;
  originalParameters : NonExecutablePresentationParameters;
  customizedFrom : Optional<RecipeIdentityAndVersion>;
  groupingHints : PresentationGroupingHints
}
```

I nomi sono descrittivi e non definiscono un formato pubblico. Un elemento di presentazione può corrispondere a più rule di una macro. `SemanticNodeAddress` è un collegamento per la UI, non un local scalar reference disponibile alle rule. Un indirizzo a una foglia interna non autorizza a estrarla dall'owner.

Proprietà richiesta:

```text
Eval(AST, contracts, evidence, T, withMetadata)
  = Eval(AST, contracts, evidence, T, withoutMetadata).
```

La recipe originaria può restare riconoscibile dopo una personalizzazione, ma non determina il significato del filtro salvato. Mappings, contratti, guardie, periodi e consumer necessari sono nell'AST o nelle definizioni semantiche effettivamente referenziate. Una recipe “ordini rimborsati” non può correggere in metadata una ratio che conta operazioni Refund.

**Riferimenti normativi (specifica 06 v1.2):** §§20–22.

---

## 18. Raccordo progettato del record filtering con Behavioral

Il catalogo **non è più sconosciuto**. Il rapporto di riconciliazione relativo a `e66eea846` ha ricostruito il modello pubblico, i 26 operatori, i tipi, i limiti e le divergenze storiche; il commit `d69e6168` ha successivamente cambiato intenzionalmente la gestione degli assenti dei 20 operatori sui valori e corretto alcune incongruenze fra executor. Il presente raccordo usa **questa nuova baseline**. La verifica sui warehouse reali per Snowflake resta non attestata.

```ml
TranslateRecord(FilterCondition(property, operator, values), schema) =
  V(PropertyTest(RecordCondition(ResolveExistingPath(property, schema),
                                 ExistingRecordOperator(operator),
                                 PrepareExistingOperands(values, schema, operator))))

TranslateRecord(And(children)) = CanonicalAnd(Map(TranslateRecord, children))
TranslateRecord(Or(children))  = CanonicalOr(Map(TranslateRecord, children))
TranslateRecord(AbsentFilter) = AllProfiles
```

Questa è una **relazione semantica concettuale**, non un formato pubblico o una proposta di modificare il persistito. Le 26 condizioni restano autonome anche per i negativi; non si devono espandere in un generico NOT o in alberi più profondi per forzarle nel catalogo vecchio. `FilterCondition.Values` contiene letterali, non riferimenti ad altre proprietà. Il confronto fra due proprietà del medesimo profilo o evento/profilo corrente è invece una capacità **nuova** con costruttore distinto.

La mappatura del filtro pubblico conserva il gruppo AND/OR alla radice. Massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. La forma canonica interna può eliminare singleton e appiattire gruppi omogenei **senza alterare** l'input pubblico, i limiti validati, il persistito o gli effetti di `Where.Equal`. Una nuova rule Behavioral occupa un nodo discendente dell'albero pubblico e mantiene il proprio budget interno.

Il registro semantico con `meaningRevision` si applica ai nuovi riferimenti governati, ai contratti e alle definizioni di evidenza secondo §3. Non promuove automaticamente i path raw esistenti a identity versionate e non cambia i filtri già salvati se un ruolo del profilo viene riassegnato. Le nuove condizioni con property reference hanno una rappresentazione esplicita, non una String magica in `values`.

**Limite dell'attestazione:** la rappresentabilità del catalogo è requisito del Canonical AST revisionato; la prova di una conversione **wire e persistita** lossless dei dati installati è distinta, ed è rimandata alla Fase 4B. La migrazione dei filtri antecedenti a `typed-filter-values` è un tema di rilascio separato, già indicato in `RELEASE-UPGRADE.md`. Non si dichiara eseguita una prova di compatibilità su filtri reali né si aggiunge una modalità `legacyMode` o `unknownPolicy`.

**Divergenze tecniche rimaste fuori da questo raccordo:** presenza/null di proprietà ordinary nullable non-Profile e Object appiattiti, collazioni del testo ordinary, Float/NaN e precisione temporale residua; Snowflake remoto non verificato. Vanno documentate e affrontate indipendentemente; la presente revisione non approva automaticamente modifiche a questi comportamenti. Per le combinazioni ammesse in Behavioral resta vincolante §14.4: le questioni aperte non autorizzano significati diversi in funzione dell’executor.

**Emendamento successivo sul testo nativo:** per i confronti Record ordinari su colonne native String e per `contains` / `does not contain` su `Array(String)` si applica la deroga esplicita approvata in §14.4.1. Le altre questioni tecniche non acquistano per questo una deroga all’invarianza.

---

## 19. Evolution model

| Extension point | Classificazione | Condizione di compatibilità |
|---|---|---|
| Nuova riduzione atomica | Closed union version extension | Definizione esplicita di input, output, unità/grana, vuoto, guardie, incertezza e consumer; non riceve altre misure |
| Nuovo scalar predicate | Closed union version extension | Operatori/types e null/missing definiti; nessun codice o generic operator string |
| Nuovo preset E | Closed union version extension | Posizioni e campi chiusi; nessun arbitrary nth-to-nth |
| Nuovo T recognizer | Closed union version extension | Parametri e bordi espliciti; non automaticamente ammesso dentro QPC |
| Nuovo QPC per-period recognizer | Closed union version extension distinta | Inserimento esplicito nel suo sotto-catalogo; niente auto-annidamento |
| Nuova istanza/mapping di un ruolo | Registry extension | Stessa firma e significato dichiarato; identity/namespace e dati registrati adeguati; ulteriori garanzie di copertura soltanto dove richieste dal contratto |
| Nuovo authoritative contract in shape già supportata | Registry extension approvata | Fonte e lifecycle mirati; non abilita slot di shape sorelle |
| Nuova famiglia con nuovo tipo di slot | Closed union version extension + registry | Richiede una nuova forma chiusa, non universal body |
| Nuovo named evidence scope | Registry extension del contratto versionato | Nome finito, stesso owner, operazioni già supportate, assenza di traversal |
| Nuovo producer per forma CDE esistente | Registry/binding extension | Soddisfa il significato e la firma già dichiarati; non muta i filtri salvati |
| Nuova forma CDE | Closed union version extension + registry | Nuovi consumer e requisiti espliciti; scalar non promosso automaticamente |
| Aumento di un limite numerico | Configuration-only extension del catalogo di limiti versionato | Stessi ruoli, stessi slot e stessa semantica; aggiornamento coerente dei bounded types |
| Nuova unità/valuta/enum | Registry extension | Nessuna conversione implicita o nuovo operatore aritmetico |

Per il filtro principale, la versione 1.2 mantiene i limiti già approvati nella 1.1: massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. Restano **2 livelli** nei body/where, **4 scope**, N<=10 e P<=5. “Configuration-only” non significa che un workspace possa disabilitare questi limiti e restare conforme alla stessa edizione. L’ampliamento rispetto alla 1.0 fu approvato nella 1.1 e non è una nuova decisione della versione 1.2; eventuali ulteriori ampliamenti richiedono approvazione e aggiornamento coerente dei bounded types, comprese le tuple enumerate quando usate.

Una revisione che cambia il significato di mapping, fonte, calendario, lifecycle o producer non viene adottata implicitamente tramite un alias “latest”. L'AST punta alla revisione riconoscibile pertinente. Non vengono definiti qui il formato di versioning o il protocollo di backward compatibility.

**Riferimenti normativi (specifica 06 v1.2):** §§22.3, 29.

---

## 20. Out-of-language proof

Il confine può essere strutturale, nominale/type-level oppure di firma registry. Quest'ultimo è necessario quando la legalità dipende dal significato di una sorgente esterna, non da una forma sintattica.

| Excluded capability | Why the AST cannot represent it | Livello decisivo |
|---|---|---|
| Arbitrary arithmetic formulas | AtomicReduction prende uno scope; ReadableAtom non contiene terminali; nessun Add/Multiply/Expression | Structural/type-level |
| Ratio di ratio; difference divisa per C | Il risultato del terminale non ha il tipo ReadableAtom | Type-level |
| Arbitrary join | Nessun join predicate; correlazione solo tramite un IdentityRole comune e owner-bound | Structural + registry per la semantica della key |
| Chiavi diverse per edge / due blocchi P | PathLayout contiene al massimo un KeyedChain con una sola K | Structural |
| Cross-condition variables | CloseOwner nasconde l'owner; Rule non riceve LocalScalar di una sibling | Type-level/lexical |
| Arbitrary relation traversal | Nessun RelationTraversal; Cart ha un solo member slot non ricorsivo; named scopes restituiscono foglie | Structural + contract shape |
| Query ricorsiva | Scope e body non contengono Rule/query; catene P limitate; nessun costruttore ricorsivo di insiemi | Structural |
| Processi ricorsivi / branching di workflow | P cerca una tupla finita; nessun process state o ramo che restituisce un altro percorso | Structural |
| Dynamic population ranking/percentile | Nessun population input o rank reduction; source sempre riferito al profilo corrente | Structural; scalar esterno già pubblicato resta distinto |
| Arbitrary nth-to-nth | E single non esporta la posizione; pair preset ha un solo field e posizioni fisse; no local scalar cross-E | Structural/type-level |
| Ordinamento per importo/score | Selector enum contiene solo posizione temporale ammessa | Structural |
| General statistics | Nessuna riduzione median/stddev/quantile/regression; producer algoritmi esterni non sono nodi | Structural |
| Arbitrary time-series expressions | T union finita; QPC non restituisce serie o scalar; nessun bucket-value reference | Structural |
| QPC annidata o AND/OR di recognizer | ClosedPerPeriodRecognizer non contiene QPC, Bool o Rule | Structural |
| General historical lookup | Nessun as-of reader; current/event-captured/certified-historical sono field semantics differenti | Structural + registry |
| Cross-profile correlation | Non esiste subject selector, otherProfile reference o account traversal; i contratti devono mantenere il subject consentito | Structural + registry |
| Generic NOT | Nessun tag Not in nessuna algebra; None applicabile soltanto ai body chiusi K | Structural |
| Complement di un P mediante K | K body non può contenere P o Root Rule | Structural |
| Workflow/live trigger | Nessun action, wait, schedule, send, consume-match o reentry node | Structural |
| Conteggio combinazioni di percorso | P produce Truth; AtomicReduction non accetta P | Type-level |
| Key lookup fuori dal blocco P | Slot keyed non costruibili per intervalli esterni; finale senza key quando suffix esiste | Type-level/structural |
| Campi dinamici del profilo in base a un evento | FieldRef è una definizione statica, non risultato di una lettura; mapping dei canali espandibile come alternative finite | Structural |
| Recupero del testimone da un derived scalar | CertifiedScalarUseRef non fornisce occurrence identity o local owner; non è OccurrenceSource | Type-level |
| Saved measure come formula inline mascherata | Nessun reference a un terminale salvato come operando; solo pubblicazione esterna certificata distinta | Structural + registry |
| Video algorithm nel percorso | P vede milestone/field normali; nessun segment range, historical-video slot o producer invocation | Structural |
| Null→zero / ultimo non-null / history fallback | Non esistono fallback constructors o strategie configurabili | Structural; contratto/evidenza devono rispettare il significato |

La lettura di un risultato esterno certificato, anche di un'analisi non inline, è consentita dove prevista. Non contraddice questa prova: il filtro può consumare un dato senza rappresentare l'algoritmo che lo produce. Non si attribuisce per questo copertura nativa al ranking, alla statistica, alla ricostruzione storica o alla copertura video.

Una OR alla root fra due P indipendenti è una composizione di risultati di profilo, non branching all'interno di un processo con testimoni condivisi. Analogamente, un predicato su un membro posseduto e il solo CartProducts slot non sono un relation graph generale.

**Riferimenti normativi (specifica 06 v1.2):** §§7, 10–18, 22–24, 27.

---

## 21. Worked mappings — 21 casi

### Convenzioni degli esempi

I simboli seguenti sono **abbreviazioni editoriali**, espanse nei costruttori dichiarati; non sono binding del filter language:

```ml
W30 = ElapsedLookback(30 elapsed days)
W90 = ElapsedLookback(90 elapsed days)
W7  = ElapsedLookback(7 elapsed days)
H   = FullRelevantHistory(RegisteredQueryableHistory@1)
Cal = FixedCalendar(EuropeRomeCalendar@1)

scope(source, grain, period, where=Unfiltered)
  = FreeBehaviorScope { population={source; grain; where}; period }

count(s)       = Reduced(Count { scope=s; sample=NoExtraMinimum })
sum(s,f)       = Reduced(Sum { scope=s; field=f; sample=NoExtraMinimum })
avg(s,f,min)   = Reduced(Average { scope=s; field=f; minimumOccurrences=min })
distinct(s,d)  = Reduced(CountDistinct { scope=s; dimension=d; sample=NoExtraMinimum })
cmp(a,op,b)    = Direct(Compare { left=a; operator=op; right=b; compatibility=inferred })
```

Ogni root rule dell'esempio è confezionata come `ScopedRule(Cal, semantic)` e un filtro non vuoto usa `Constrained(Leaf(rule))`, salvo S012 che mostra la congiunzione. Il default del filtro è Europe/Rome. Le espressioni `SameIdentity(...)`, `SameObservedValue(...)`, `Self` e `Named(...)` sono consentite **soltanto dentro il rispettivo owner** e non sono variabili condivisibili.

**Parametri non esplicitati nelle frasi:** si usa W90 nei casi monetari e di ripetizione senza periodo, W30 per view/comportamenti di interesse e opportunità, W7 per il ritorno. Sono scelte illustrative, non inferenze dalla specifica né default automatici. Nei casi 13–14 la soglia 100 viene resa come €100 e il requisito monetario EUR resta nello stadio mostrato. Un raw `Number` richiederebbe invece il relativo literal numerico, senza che il nome `amount` ne certifichi il significato monetario.

`paidComparableEUR` è un campo dichiarato confrontabile in EUR che conserva la popolazione richiesta; il modello non elimina ordini di altre valute per sommarlo. I riferimenti `@1` sono segnaposto di definizioni semantiche versionate esemplificative, non attestazioni di producer o mapping realmente installati.

### 21.1 Country = IT

```ml
V(PropertyTest(EqualLiteral {
  field=CurrentProfile.country@1;
  op=Equal;
  value=Literal<CountryDomain>(IT)
}))
```

**Primitiva:** V. **Contesti locali:** nessuno. **Registry:** proprietà corrente country, tipo/normalizzazione CountryDomain. **Invariante principale:** proprietà corrente a T e literal dello stesso dominio; non una stringa normalizzata arbitrariamente.

### 21.2 Purchase almeno 3 volte negli ultimi 30 giorni

```ml
V(cmp(
  count(scope(Purchase@1, OrderGrain@1, W30)),
  GreaterOrEqual,
  Literal<Count<OrderGrain>>(3)
))
```

**Primitiva:** V. **Contesti locali:** nessun contextual owner. **Registry:** Purchase, order grain, dedup identity e observation contract. **Invariante:** tre ordini/azioni business distinte; tre copie dello stesso evento non bastano.

### 21.3 Purchase EUR >€100 sulla stessa occurrence

```ml
V(cmp(
  count(scope(Purchase@1, OrderGrain@1, W90,
    Where(RequireCurrencyAndTest {
      money=NativePaidAmount@1;
      currency=EUR@1;
      test=GreaterThan(MoneyLiteral<EUR>(100))
    }))),
  GreaterOrEqual,
  CountLiteral(1)
))
```

**Primitiva:** V. **Contesti locali:** nessun owner contestuale; un solo same-occurrence scope. **Registry:** Purchase, grain ordine, campo monetario/discriminante di valuta, EUR. **Invariante:** valuta e importo si applicano allo stesso Purchase, non a due scope combinati alla root.

### 21.4 Nessun Purchase EUR negli ultimi 30 giorni

```ml
V(cmp(
  count(scope(Purchase@1, OrderGrain@1, W30,
    Where(EqualLiteral(NativeCurrency@1, Equal, EUR)))),
  Equal,
  CountLiteral(0)
))
```

**Primitiva:** V. **Contesti locali:** nessuno. **Registry:** Purchase, currency, grain/identity e interpretabilità dei dati registrati; nessun certificato esterno di coverage. **Invariante:** una currency ignota è possibile evidence di Purchase EUR; non viene scartata per provare zero. Nessuna guardia di non-vuoto viene aggiunta all'assenza.

Il riferimento `NativeCurrency@1` è governato: il caso con null business sconosciuto e il distinto confronto Record raw su JSON Null sono specificati nella specifica 06 v1.2, §28.20. Questo esempio non autorizza la sostituzione automatica fra le due richieste.

### 21.5 Spesa totale >€1.000

```ml
V(cmp(
  sum(scope(Purchase@1, OrderGrain@1, W90), paidComparableEUR@1),
  Greater,
  MoneyLiteral<EUR>(1000)
))
```

**Primitiva:** V. **Contesti locali:** nessuno. **Registry:** total paid meaning, proprietà additiva confrontabile EUR, order grain. **Invariante:** la popolazione non viene ristretta ai soli ordini con nativeCurrency=EUR; nessuna conversione implicita. Sum certamente vuota vale zero EUR.

### 21.6 Average order value >€100 con almeno 2 ordini

```ml
V(cmp(
  avg(scope(Purchase@1, OrderGrain@1, W90), paidComparableEUR@1, min=2),
  Greater,
  MoneyLiteral<EUR>(100)
))
```

**Primitiva:** V. **Contesti locali:** nessuno. **Registry:** campo medio nella stessa unità, order grain e popolazione. **Invariante:** il minimo 2 è nella stessa riduzione/test, non una sibling da cui leggere evidenza. Un ordine da €500 non soddisfa il requisito; un importo mancante non è ignorato dalla media.

### 21.7 Almeno 5 prodotti distinti

```ml
V(cmp(
  distinct(
    scope(Purchase@1, ProductPerOrderGrain@1, W90),
    ProductDimension@1),
  GreaterOrEqual,
  Literal<Count<DistinctValues<Product>>>(5)
))
```

**Primitiva:** V. **Contesti locali:** nessuno. **Registry:** proiezione prodotto-per-ordine, dimensione Product e suo dominio/uguaglianza. **Invariante:** distinti Product, non line item né quantità acquistate. La frase è istanziata qui sui prodotti acquistati in W90.

### 21.8 Purchase count > Refund-operation count

```ml
V(cmp(
  count(scope(Purchase@1, OrderGrain@1, W90)),
  Greater,
  count(scope(Refund@1, RefundOperationGrain@1, W90))
))
```

**Primitiva:** V. **Contesti locali:** nessuno; due scope atomici indipendenti. **Registry:** i due repertori/grane e la compatibilità numerica dei count. **Invariante:** type compatibility non richiede grane identiche, ma non cancella la differenza tra ordini e operazioni Refund.

### 21.9 Refund-operation-count / order-count <10%

```ml
V(Ratio {
  numerator=count(scope(Refund@1, RefundOperationGrain@1, W90));
  denominator=count(scope(Purchase@1, OrderGrain@1, W90));
  compatibility=CountRatioCompatibility;
  test=LessThan(DimensionlessLiteral(1/10))
})
```

**Primitiva:** V. **Contesti locali:** nessuno. **Registry:** entrambi i repertori, grane e periodi. **Invariante:** denominatore positivo intrinseco; il risultato non è limitato a 100% e può contare più Refund sullo stesso ordine o Refund di ordini precedenti.

### 21.10 Share di ordini con refund <10%

```ml
K(CorrelatedEventObjects {
  candidates={ primary=scope(Purchase@1, OrderGrain@1, W90);
               keyRole=OrderIdentityRole@1; access=PurchaseOrderIdentityAccess };
  primaryAccess=IdentityOnly;
  verify=Leaf(SameIdentityValueTest(
    cmp(count(SameIdentity(Refund@1, RefundOperationGrain@1, H)),
        GreaterOrEqual, CountLiteral(1))));
  quantify=Share { minimumCandidates=1; test=LessThan(FractionLiteral(1/10)) }
})
```

**Primitiva:** K. **Contesti locali:** un order candidate owner; nessun occurrence handle esportato. **Registry:** Order identity comune a Purchase e Refund, seme degli ordini, storia/horizon dei rimborsi. **Invariante:** ogni ordine candidato contribuisce al massimo una volta al numeratore e una volta al denominatore; il body non cambia la base.

**Differenza fra 9 e 10.** Il caso 9 contiene due operandi atomici in una Ratio V. Il caso 10 contiene una base di identità ordine, un body correlato e un quantificatore Share K. Nella fixture normativa con dieci ordini e due Refund sullo stesso ordine, i valori sono 20% e 10%; alla soglia stretta `<10%` nessuno dei due passa, pur essendo diverse le quantità. In una fixture illustrativa con dodici ordini e due Refund su uno solo, 2/12 non passa `<10%`, mentre 1/12 passa. Nell'esempio 10 l'orizzonte dei Refund è fino a T per gli ordini della base: anche questo resta esplicito e non viene derivato dal periodo del numeratore di 9.

### 21.11 Stesso prodotto acquistato in almeno 3 ordini distinti

```ml
K(ObservedValues {
  candidates={ seed=scope(Purchase@1, ProductPerOrderGrain@1, W90);
               dimension=ProductDimension@1 };
  verify=Leaf(SameRepertoireMeasureTest(
    cmp(distinct(SameObservedValue(W90), OrderDimension@1),
        GreaterOrEqual, CountLiteral(3))));
  quantify=Some
})
```

**Primitiva:** K observed value. **Contesti locali:** un Product value owner, vincolato al repertorio Purchase. **Registry:** Product dimension, proiezione prodotto-per-ordine e Order distinct dimension. **Invariante:** la misura interna conta ordini distinti, non unità o righe. Non viene creato un secondo insieme navigabile di ordini.

### 21.12 Esiste prodotto visualizzato ma mai acquistato

```ml
K(CorrelatedEventObjects {
  candidates={ primary=scope(View@1, ViewActionGrain@1, W30);
               keyRole=ProductIdentityRole@1; access=ViewProductIdentityAccess };
  primaryAccess=IdentityOnly;
  verify=Leaf(SameIdentityValueTest(
    cmp(count(SameIdentity(Purchase@1, OrderGrain@1, H)),
        Equal, CountLiteral(0))));
  quantify=Some
})
```

**Primitiva:** K correlated event object. **Contesti locali:** Product identity owner. **Registry:** product role comune fra View e Purchase, inclusa eventuale associazione multivalore nell'ordine, full relevant history. **Invariante:** nessun vincolo “dopo la view” è aggiunto. Un acquisto precedente della stessa Product identity esclude quel candidato.

### 21.13 L'ultimo Purchase ha amount >100

```ml
E(SingleExperience {
  base=scope(Purchase@1, OrderGrain@1, H, Unfiltered);
  select=Last;
  verify=Leaf(SelectedScalarPredicate(
    RequireCurrencyAndTest {
      money=Self.NativePaidAmount@1;
      currency=EUR@1;
      test=GreaterThan(MoneyLiteral<EUR>(100))
    }))
})
```

**Primitiva:** E. **Contesti locali:** selected Purchase owner. **Registry:** Purchase, order grain, action ordering e money/discriminator. **Invariante:** si sceglie prima l'ultimo Purchase della base; EUR e importo sono verificati dopo. L'unità EUR è l'istanziazione esplicita della soglia monetaria omessa nella frase.

### 21.14 L'ultimo Purchase fra quelli EUR ha amount >100

```ml
E(SingleExperience {
  base=scope(Purchase@1, OrderGrain@1, H,
    Where(EqualLiteral(NativeCurrency@1, Equal, EUR)));
  select=Last;
  verify=Leaf(SelectedScalarPredicate(
    RequireCurrencyAndTest {
      money=Self.NativePaidAmount@1;
      currency=EUR@1;
      test=GreaterThan(MoneyLiteral<EUR>(100))
    }))
})
```

**Primitiva:** E. **Contesti locali:** selected Purchase owner distinto da quello dell'esempio 13. **Registry:** gli stessi riferimenti monetari e selettivi. **Invariante:** il filtro EUR è nella selection base; amount rimane nel verify. La guardia di valuta può essere staticamente riconosciuta come già richiesta dalla base, senza spostare altri predicati.

Con EUR €150 seguito da USD, 13 non ripiega sul primo ordine; 14 sceglie il Purchase EUR. Con EUR €150 seguito da EUR €40, entrambi falliscono: nessuno seleziona “l'ultimo >100”.

### 21.15 View THEN Purchase same product

```ml
P(OrderedPath {
  layout=Correlated(Whole(KeyedChain {
    keyRole=ProductIdentityRole@1;
    steps=[
      { base=scope(View@1, ViewActionGrain@1, W30); select=AnyOccurrence },
      { base=scope(Purchase@1, OrderGrain@1, W30); select=AnyOccurrence }
    ]
  }));
  adjacentDurations=NoExtraDurations;
  totalDuration=None;
  negativeChecks=NoNegativeRequirements;
  finalOutcome=None
})
```

**Primitiva:** P. **Contesti locali:** un path owner, due step slot, un blocco Product sull'intero percorso. **Registry:** View/Purchase, Product key role e accesso eventuale multivalore nell'ordine, grane e ordine. **Invariante:** due occurrence distinte ordinate con un valore Product comune; non basta View A e Purchase B.

### 21.16 Purchase same-order Refund THEN new Purchase

```ml
P(OrderedPath {
  layout=Correlated(Starts {
    block=KeyedChain {
      keyRole=OrderIdentityRole@1;
      steps=[
        { base=scope(Purchase@1, OrderGrain@1, W90); select=AnyOccurrence },
        { base=scope(Refund@1, RefundOperationGrain@1, W90); select=AnyOccurrence }
      ]
    };
    bridge=OrdinaryPrecedence;
    suffix=PlainChain [
      { base=scope(Purchase@1, OrderGrain@1, W90); select=AnyOccurrence }
    ]
  });
  adjacentDurations=NoExtraDurations;
  totalDuration=None;
  negativeChecks=NoNegativeRequirements;
  finalOutcome=None
})
```

**Primitiva:** P. **Contesti locali:** path owner; blocco Order sui passi 1–2; passo 3 unkeyed. **Registry:** Purchase/Refund e Order role nei primi due passi, non nel terzo. **Invariante:** “new Purchase” richiede una nuova occurrence successiva, non lo stesso order ID e neppure un order ID diverso per vincolo implicito.

La specifica O03 aggiunge “primo Purchase di sempre”: la relativa variante cambia **soltanto** il primo passo in `base=Purchase(H), select=FirstInBase`, con periodi degli altri passi dichiarati pertinenti. La selezione resta sulla base del profilo, non per order candidate. L'esempio qui richiesto non conteneva “primo”, quindi non lo aggiunge silenziosamente.

### 21.17 Messaggio non cliccato entro 7 giorni

```ml
K(Opportunities {
  base={ origin=FirstMessageDelivery(MessageOpportunity@1);
         startPeriod=W30;
         startQualifiers=Unfiltered;
         maturityRestriction=AllOpportunities };
  select=AllInBase;
  verify=Leaf(Outcome {
    behavior=ClickedOutcomeForThisMessage@1;
    horizon=WithinAfterStart(7 elapsed days);
    requirement=DidNotOccur
  });
  quantify=Some
})
```

**Primitiva:** K opportunity. **Contesti locali:** message opportunity owner e il suo start. **Registry:** prima consegna registrata identificabile, identity/start, Clicked attribuito allo stesso messaggio e ordine dei dati disponibili; nessun prefisso esterno certificato. **Invariante:** deadline inclusiva e maturazione T>start+7d; un messaggio immaturo rimane candidato con esito pendente. Un click reale all'ottavo giorno non confuta l'assenza entro sette.

### 21.18 Ticket attualmente aperto da oltre 48h senza Human Response

```ml
K(AuthoritativeObjects(NamedEvidenceObjects {
  contract=TicketCurrent@1;
  candidateFilter=Unfiltered;
  verify=And [
    CurrentObjectPredicate(EqualLiteral(Self.currentStatus@1, Equal, Open)),
    LifecyclePredicate(AgeOfInitialOpening(Greater, 48 elapsed hours)),
    NamedEvidenceValueTest(cmp(
      count(Named(HumanResponse@TicketCurrent1,
                  horizon=InitialOpeningInclusiveUntilT,
                  where=Unfiltered)),
      Equal, CountLiteral(0)))
  ];
  quantify=Some
}))
```

**Primitiva:** K authoritative, shape NamedEvidence. **Contesti locali:** un Ticket owner; una sola foglia comportamentale named. **Registry:** inventario e stato Ticket, lifecycle identity, initial openedAt, Human Response con horizon/operazioni/coverage. **Invariante:** stato, età strettamente >48h e risposte appartengono allo stesso ticket lifecycle. La riapertura non azzera l'età e una risposta su un altro ticket è irrilevante.

### 21.19 Attivo almeno 2 giorni in almeno 3 delle ultime 4 settimane complete

```ml
T(QualifiedPeriodCardinality {
  population={ source=Usage@1; grain=UsageActionGrain@1; where=Unfiltered };
  periods=LastComplete { kind=Week; count=4 };
  recognize=AtLeastDistinctActiveDays(2);
  quantify=AtLeast(3)
})
```

**Primitiva:** T. **Contesti locali:** nessun owner leggibile; i bucket sono posizioni interne non esportabili. **Registry:** Usage/grain, calendario Europe/Rome e observation contract. **Invariante:** quattro periodi fissi, ognuno con almeno due giorni distinti per qualificare; la quarta settimana vuota resta nella serie. Non è un count di sei giorni complessivi.

### 21.20 Tornato dopo almeno 30 giorni di inattività

```ml
T(ResumedAfterInactivity {
  returns=FromActivity {
    population={ source=Usage@1; grain=UsageActionGrain@1; where=Unfiltered };
    minimumInactivity=30 elapsed days
  };
  returnScope=ReturnTimeIn(W7)
})
```

**Primitiva:** T. **Contesti locali:** episodio canonico interno; nessuna ancora esportata perché non si usa LastQualifiedReturn. **Registry:** Usage/repertorio, action identity/order e tempi dei record; pausa senza intermedi registrati, con predecessore registrato. **Invariante:** r è un ritorno preceduto da un uso reale u senza intermedi; u può stare prima di W7. Il primo utilizzo assoluto non passa.

### 21.21 Path sulle milestone certificate di S012

Il path è P; il segmento completo S012 è AND di V, K e P. La seguente abbreviazione di reference identifica evidenza già certificata, non una chiamata a un algoritmo inline:

```ml
M1 = CertifiedMilestoneUse {
  definition=FirstThresholdAttainmentFromWindowStart@1;
  parameters={ content=SeriesA.Episode1VersionRef;
               window=W30; threshold=FractionLiteral(9/10) };
  permittedUse=PathStepCapability
}
M2 = CertifiedMilestoneUse {
  definition=FirstThresholdAttainmentFromWindowStart@1;
  parameters={ content=SeriesA.Episode2VersionRef;
               window=W30; threshold=FractionLiteral(9/10) };
  permittedUse=PathStepCapability
}

And [
  V(PropertyTest(EqualLiteral(CurrentProfile.language@1, Equal, IT))),

  K(AuthoritativeObjects(StateOnlyObjects {
    contract=CurrentSubscription@1;
    candidateFilter=Unfiltered;
    verify=Leaf(CurrentObjectPredicate(PayingActiveAtT));
    quantify=Some
  })),

  P(OrderedPath {
    layout=Correlated(Whole(KeyedChain {
      keyRole=SeriesIdentityRole@1;
      steps=[
        { base=scope(M1, CertifiedAttainmentGrain@1, W30);
          select=AnyOccurrence },
        { base=scope(M2, CertifiedAttainmentGrain@1, W30,
            Where(And [
              ElapsedAgeTest(GreaterOrEqual, 7 elapsed days),
              EqualLiteral(NextEpisodeAvailableAtThisMilestone@1, Equal, true)
            ]));
          select=AnyOccurrence }
      ]
    }));
    adjacentDurations=NoExtraDurations;
    totalDuration=None;
    negativeChecks=NoNegativeRequirements;
    finalOutcome=SameKeyOutcomeFromFinal {
      behavior={ source=PlaybackStarted@1; grain=PlaybackStartGrain@1;
                 where=Where(EqualLiteral(EpisodeIdentity@1, Equal, SeriesA.Episode3Ref)) };
      horizon=AfterFinalUntilT;
      requirement=DidNotOccur
    }
  })
]
```

**Primitive:** P per il percorso; V/K/P per il segmento completo. **Contesti locali:** un Subscription candidate owner indipendente e un path owner con due milestone e una sola Series key; nessuna key passa tra K e P. **Registry:** proprietà lingua; Subscription corrente; definizione e parametri delle milestone; content/version e Series identity; proprietà storica certificata sulla seconda milestone; PlaybackStarted ed Episode identity; ordine/coverage.

**Invarianti principali:** la definizione di M1/M2 certifica il primo raggiungimento del 90% in W30, quindi `AnyOccurrence` non autorizza una rivisione successiva sostitutiva. Il vincolo di età include esattamente sette giorni. La disponibilità di Episode3 è un fatto storico della seconda milestone, non lo stato corrente di un contenuto. Il finale può leggere Series perché entrambi i passi costituiscono un blocco correlato di almeno due passi e il finale ne fa parte.

Il blocco sulla Series non è un requisito aggiuntivo nascosto: le due content reference identificano già episodi della stessa Series A e l'esito riguarda il suo episodio 3. È una rappresentazione esplicita della stessa identity pertinente. Non si introducono key differenti episode/order né navigazioni Series → Episodes nell'AST.

Se la prima milestone di Episode2 precede quella di Episode1, la tupla non viene recuperata con un secondo raggiungimento. Se manca il producer conforme, la costruzione è valida come capability del linguaggio ma non attivabile nel workspace; se l'evidenza storica pertinente è incerta per il profilo, può produrre UNKNOWN. Il producer video non è una capacità obbligatoria del core.

**Riferimenti per i mapping (specifica 06 v1.2):** §§10–17 e acceptance §§28.2–28.18, in particolare §§28.11/S012, 28.13, 28.14, 28.15 e 28.16.

---

## 22. Alternative tecniche scartate

| Decisione | Alternative considerate | Scelta e ragione tecnica |
|---|---|---|
| Local context | Pure ownership senza riferimenti; handle nominali; local IDs generici; de Bruijn indices | Ownership strutturale e Self implicito per quasi tutti gli usi; handle nominali solo per slot P finiti. Gli ID generici e de Bruijn renderebbero sintatticamente naturali riferimenti a ruoli non autorizzati; richiederebbero comunque una seconda algebra di permessi |
| K architecture | `IdentitySet(source,key,body:AnyExpression)`; un tipo per ogni recipe; union per famiglie semantiche | Union Observed/Correlated/Opportunity/Authoritative e shape autorevoli ulteriormente distinte. Evita sia universal body sia proliferazione di primitive/recipe |
| P correlation | Array di step + start/end; lista di edge equalities; prefix/block/suffix | Un solo blocco strutturale. Start/end è controllabile ma non incorpora contiguità e unicità nella forma; edge equalities aprono più chiavi e intersezioni pairwise |
| Arithmetic | Expression AST; un generico `Measure`; atomic reads + terminal tests | Atomic/terminal disgiunti. Non serve vietare addizione o ratio ricorsiva con regole supplementari: mancano i costruttori e i tipi |
| Time patterns | TimeSeriesExpression; generic bucket body; closed recognizers | T chiuso, QPC con sotto-union di sei recognizer non ricorsivi. Le quattro componenti population/series/recognizer/cardinality restano separate |
| Semantic references | Raw string per tutto; ontology obbligatoria; typed refs con percorso raw | Typed refs versionati, raw schema ammesso. Non si scambia business meaning con display label o raw path |
| Object access | RelationTraversal unico; body generico con allowlist | Shape StateOnly/Named/Cart/Trial. Le differenze object-correlated/member/temporal non sono flag su una relazione universale |
| Certified evidence | `EvidenceRef<any>`; tutti i derivati come profile scalar | Forme e consumer nominalmente distinti; scalar, occurrence, milestone, current state e solo episodio di ritorno ammesso mantengono il rispettivo significato |
| Guards | Albero booleano arbitrario di guardie; predicate root separati | Guardie intrinseche o parametri del costruttore/operando. Una recipe non perde il proprio minimo e le guardie non diventano valori |
| Singleton groups | Conservarli per corrispondere alla UI; eliminarli | Eliminati dal canonical AST; la struttura UI rimane in provenance |
| Canonical equality | Uguaglianza testuale/wire; equivalenza logica completa | Uguaglianza strutturale normalizzata con reference di significato immutabile. Indipendente da JSON e realizzabile senza inferenze simboliche generali |

Le alternative sono confrontate soltanto come rappresentazioni del linguaggio della specifica 06 v1.2. Non sono proposte nuove capability, nuovi modelli di prodotto o una nuova architettura esecutiva.

---

## 23. Internal consistency review

Questa è una review strutturale del modello, non l'esito di test di un motore o di compilazione in uno specifico linguaggio di programmazione. I seguenti tentativi devono essere rifiutati dai tipi/costruttori indicati.

| Test | Tentativo | Confine verificato |
|---|---|---|
| A — Arithmetic safety | `Ratio(Ratio(A,B),C)` | Ratio interno è TerminalTest; lo slot esterno richiede ReadableAtom |
| B — Cross-condition safety | Una sibling legge Self.order di un'E precedente | CloseOwner elimina quel token dall'ambiente root; LocalScalar<Root> è Never |
| C — Correlation safety | Due blocchi o due key nello stesso P | Nessun CorrelatedLayout contiene due KeyedChain; K unico |
| D — K slot safety | Ticket usa CartProducts o TrialUpgrade | NamedEvidenceObjectLeaf non contiene quei costruttori; un ref di altro owner non ha il tipo richiesto |
| E — Temporal safety | QPC(QPC(...)) o QPC(AND(recognizer1,recognizer2)) | Il recognizer non contiene né QPC né Bool |
| F — Evidence safety | CertifiedScalar usato come P step grazie a timestamp metadata | OccurrenceSource non contiene CertifiedScalarUseRef |
| G — Selection safety | `Last(all Purchase) verify EUR` fuso con `Last(Purchase where EUR)` | Where di base e verify hanno posizioni distinte; nessuna riscrittura li attraversa |
| H — Domain safety | Product ID confrontato con Order ID perché entrambi string | Identifier domain/namespace nominali, nessuna coerzione implicita |
| I — Count compatibility | Order count confrontato con Refund-operation count | Ammesso da Count comparison compatibility senza cancellare le grane |
| J — Share safety | Secondo periodo o secondo source nella part | OccurrenceShare contiene un solo whole; part è solo predicate |
| K — P selection safety | First per oggetto introdotto dal blocco | PathStep base è profile-scoped; la key è imposta alla tupla dopo la selezione |
| L — P negative scope safety | Finale sul suffix legge Order del block | Starts/Middle hanno final slot WithoutKey |
| M — Same occurrence safety | Camera su una riga e quantity>=2 su un'altra | Un solo OwnedMemberTest con body riferito al medesimo membro |
| N — Budget safety | Cinque foglie identiche nascoste da dedup/hash sharing | Il budget conta usi prima dell'eliminazione e non distingue per hash |
| O — Deadline safety | Assenza certa a T=deadline | Within impone T>deadline per maturazione negativa |
| P — QPC empty-period safety | Settimana vuota eliminata o sostituita | CompleteCalendarSeries è fissa prima del recognizer |
| Q — Return safety | Primo utilizzo registrato trattato come ritorno | CanonicalReturnDefinition richiede predecessore registrato e pausa senza intermedi registrati; i contratti certificati restano distinti |
| R — Current/historical safety | NextEpisodeAvailableNow usato come fatto alla milestone | Field historical meaning diverso; il contratto di S012 richiede quello storico |
| S — Singleton/no-condition safety | AND vuoto equiparato ad AllProfiles | Costruttori distinti; empty group invalido |
| T — Provenance safety | Eliminazione del recipe name cambia il risultato | Recipe non appartiene a Rule né alla risoluzione dei semantic operands |

### 23.1 Controlli positivi di copertura degli slot più sensibili

| Caso | Forma naturale presente |
|---|---|
| S002 Cart corrente con prodotto rivisto | CurrentCarts → HasCurrentProduct → SameProductValueTest; due owner al massimo; activity legata al membro di quel Cart |
| S004 Trial | CurrentDesignatedTrial; Usage attribuito al trial per Import/Dashboard/Share; UpgradeSinceStartTest indipendente da trial ID; quattro scope |
| S011 Ultimo ritorno | T.LastQualifiedReturn con recency e mancata consegna dopo Self; somma minuti W7 in una V indipendente |
| I037 Primo seguito tardivo | E.Single con FirstFollowing; assenza del seguito non diventa ritardo infinito |
| I050 Reconciliation | K correlated Order con local paid e Sum same-order Refund; nessuna ratio o positività del pagato aggiunta |
| I059 Giorni distinti per canale | K observed channel con CountDistinct(CalendarDayDimension); la dimensione di riduzione non diventa un secondo K |
| I071 Prima azione successiva | P ordinato con assenza del repertorio UserAction nell'intervallo adiacente e periodi dei passi separati |
| I093 Ultimi due intervalli | E.LastTwoIntervals, ultime tre occurrence, due durate positive native |
| I099 Ultime tre campagne | K.Opportunities, FirstCampaignDeliveryToProfile, LatestN(3), outcome attribuito alla campagna |
| I100 Autorizzazione corrente del canale | Alternative finite per canale con field correnti fissi; nessun dynamic property lookup |

### 23.2 Limiti dichiarati della consegna

I094 (cadenza statistica), I097/S015 (completamento degli assegnati) e la ricostruzione di I098 da cronologia separata restano deferred. Uno snapshot storico I098 già certificato rimane filtrabile, senza rendere incluso il producer generale.

Il modello non attesta implementazione delle primitive E/K/P/T, disponibilità di workspace, esecuzione dei nuovi acceptance test o correttezza degli algoritmi di producer. Le disponibilità core, con contratti mirati e con derived evidence restano distinte. Il catalogo Record è noto e deve essere rappresentabile senza perdita semantica; **non** si attesta per questo una migrazione pubblica/persistita universale.

Il modello finale seguente è il riepilogo tecnico della **versione 1.2 approvata**, nuova baseline tecnica. Conserva costruttori e limiti della 1.1 con la nuova semantica del dominio registrato; non contiene scelte wire/API o un’architettura esecutiva.

---

### 23.3 Stato della riconciliazione e verifiche ancora richieste

Questa revisione documentale 1.2 è **la baseline tecnica approvata**. La 1.1 rimane riferimento storico. La baseline Record resta `d69e6168`; i test Go/PostgreSQL riportati dalla riconciliazione sono storici e non verifiche di questa revisione documentale. Snowflake reale resta non attestato. Il rapporto di revisione 1.2 documenta analisi e controlli documentali, non test di primitive implementate.

Per la Fase 4B restano da scegliere tag/payload wire, persistenza/versioning, compatibilità dei filtri installati, binding raw/governato, equality/hash e contratto fisico/esecutivo. IR/pubblicazione, acquisizione/attribuzione e matching restano lavori autonomi secondo §14.6; non sono bloccanti del progetto corrente e possono essere prerequisiti di rilascio. Ordinary nullable/Object appiattiti, collazioni, Float/NaN e precisione temporale rimangono questioni separate. Il rapporto 12 non è approvato e non viene modificato da questa revisione.

### 23.4 Delta mirato della versione 1.2

Le dichiarazioni ADT, i payload e i limiti strutturali della 1.1 restano invariati. Cambiano il dominio delle occurrence, la semantica dei descrittori storici e i requisiti di evidenza: vuoto registrato non richiede completezza esterna; dati esistenti, deadline e contratti restano rilevanti. Il binding H negli esempi usa il nuovo identificativo illustrativo RegisteredQueryableHistory@1, senza reinterpretare una definizione immutabile precedente. La semantica 1.2 è approvata; l’adozione di nuove definizioni governate e l’eventuale compatibilità di documenti Behavioral precedenti richiedono una scelta esplicita; non si alterano filtri Record o fingerprint esistenti. Le fixture CW01–CW21 e i confini del benchmark sono in 06 §§28.21–28.22. La revisione indipendente successiva introduce il vincolo dei due estremi finiti per MaximumGap, la scelta normativa della quantificazione K composizionale e la rilevanza degli ignoti sulla verità dei test di prevalenza, con fixture N1–N3 in 06 §§28.23–28.25. Non sono nuovi costruttori o capacità del linguaggio; queste correzioni N1/N2/N3 sono approvate nella versione 1.2. Nessun test di esecuzione viene attestato.

# Canonical AST — Phase 4A Final Model

**Documento estraibile — versione 1.2 approvata.** Base normativa: [Specifica semantica riconciliata — versione 1.2, approvata](06-specifica-semantica-riconciliata-v1.2.md). La 1.2 è la nuova baseline normativa e tecnica; la 1.1 del commit `25f639d84eb35815af5d1062c86545b65b97d720` è conservata come riferimento storico. Questo riepilogo mantiene costruttori e limiti strutturali con la semantica a mondo chiuso e le correzioni N1/N2/N3 approvate. Notazione ML-like, non JSON né field pubblici; nessuna scelta API, persistenza, migrazione o piano esecutivo è approvata per implicazione.

## F1. Top-level ADT e booleani

```ml
type Rule =
  | V of TypedComparison<Root>
  | E of SelectedExperience
  | K of IdentitySet
  | P of OrderedPath
  | T of ActivityPattern

type ScopedRule = { calendar : CalendarChoice; semantic : Rule }
type Filter = {
  defaultCalendar : CalendarDefinitionRef;
  root : AllProfiles | Constrained of CanonicalBool<4, ScopedRule>
}

type CanonicalBool<0, L> = Leaf of L
type CanonicalBool<d, L> where d>0 =
  | Leaf of L | And of AndGroup<d, L> | Or of OrGroup<d, L>
type AndGroup<d, L> = AtLeast2<Leaf<L> | OrGroup<d-1, L>>
type OrGroup<d, L>  = AtLeast2<Leaf<L> | AndGroup<d-1, L>>
type AndGroup<0, L> = Never
type OrGroup<0, L> = Never
```

Gli unici risultati semantici sono MATCH, NO_MATCH e UNKNOWN. Le rule leggono il medesimo profilo corrente e T. Soltanto MATCH include il profilo nella platea Profiles; MATCH→UNKNOWN rimuove la membership corrente. Failure tecnico non è un truth value.

AllProfiles rappresenta il **filtro pubblico assente**, non un gruppo vuoto. La radice pubblica, quando esiste, è AND/OR; empty group è invalido; singleton e gruppi omogenei consecutivi sono validi nel documento pubblico. Massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. La grammatica **interna** può eliminare singleton e appiattire gli operatori uguali, ma non modifica il documento pubblico, la sua persistenza o l’uguaglianza esistente. Non esiste NOT generico.

**Raccordo del riepilogo:** le cinque varianti top-level appartengono all’AST di Behavioral; il DTO pubblico rimane `core.Filter` e `FilterCondition` per le condizioni Record. Le 26 condizioni sono property test V e foglie ammesse nei body solo con le relative capability. La radice pubblica resta un gruppo AND/OR, anche se il canonical AST può usare una foglia dopo normalizzazione.

## F2. Shared types, predicate, riferimenti e tempo

### Tipi e riferimenti

```ml
type ScalarType =
  | Text<TextDomain> | Number<NumericDomain> | Boolean
  | Date<DateDomain> | DateTime<TemporalDomain> | Enumeration<EnumRef>
  | Identifier<IdentifierDomainRef, NamespaceRef>
  | Money<MoneyDomainRef, CurrencyRef>
  | Quantity<QuantityDomainRef, UnitRef>
  | Duration | Count<CountMeaning>

type CountMeaning = Occurrences<GrainRef>
                  | DistinctValues<DimensionMeaning>
                  | Candidates<CandidateGrain>
type PropertyShape = Scalar<ScalarType> | DeclaredSet<ScalarType>

opaque type Ref<Kind, Signature>          (* ID/namespace/revisione di significato immutabile *)
opaque type Proof<Requirement>            (* prova di compatibilità governata da registry *)
opaque type RecordAdmissibility<Schema, Operator, Operands>  (* validazione record esistente *)
opaque type FieldRef<Schema, ValueType>
opaque type GrainProjectionRef<Schema, Grain>
opaque type IdentityRoleAccess<Schema, Grain, Domain, Namespace>
opaque type Literal<ValueType>
opaque type PositiveInt                  (* >=1 *)
opaque type Nat                          (* >=0 *)
opaque type Position1To10
opaque type PositiveDuration
opaque type Fraction0To1
opaque type SignedFractionLiteral
opaque type SignedDeltaLiteral<Unit>
```

Sono governati con reference distinti profile property, raw behavior/property, semantic event role, repertoire, grain, semantic property, identity role, unit/currency, calendar, authoritative object contract, named evidence scope, member/temporal slot e Certified Derived Evidence definition. Display label e physical binding sono separati dall'identità semantica.

Esiste un percorso raw tramite schema tipizzato/versionato, senza ontology obbligatoria. **Per una `RecordCondition` il path è risolto contro lo schema esistente e attestato da `RecordAdmissibility`, non da `Proof` del registro governato**; un semplice `country is "IT"` non richiede nuovi prerequisiti semantici. Il raw schema non certifica da solo un ruolo business. Product ID e Order ID, namespace diversi, EUR/USD e quantità di dimensioni diverse non sono implicitamente compatibili. I count con grane diverse possono essere confrontati e usati in una ratio, mantenendo entrambe le grane. Sono necessari witness statici di compatibilità specifici dell'operazione; non un cast numerico generale.

### Predicate chiusi

`ScalarPredicate<S>` comprende `RecordCondition` (i **26 operatori del record filtering** con tipi, payload e validazione di `d69e6168`) e i nuovi predicati governati chiusi: equality/difference con literal o current-profile scalar compatibile; confronto ordinato/intervallo; membership; missing/present/explicit null/known; empty text; contains/starts/ends; appartenenza nei set; test monetario e temporale. Il record filtering conserva anche String ordinabile, Time/Year, JSON dinamico, contenitori e negativi autonomi; i nuovi predicati non ne restringono il catalogo. Nessun NOT generico né conversione implicita dei literal in property reference.

Su JSON Null presente, il confronto Record raw `properties.currency is "EUR"` con operando JSON String `"EUR"` produce FALSE. Il confronto governato `NativeCurrency = EUR`, con contratto che interpreta il null come valuta business sconosciuta, produce UNKNOWN. Con fonte e contratto validi e disponibili e un solo Purchase nel periodo completamente osservato, «nessun acquisto EUR» produce rispettivamente MATCH e UNKNOWN (specifica 06 v1.2, §28.20); le due costruzioni non sono intercambiabili automaticamente.

`RequireCurrencyAndTest(moneyField,currency,test)` lega la valuta al campo monetario tramite il mapping registrato e mantiene nello stesso stadio il requisito di valuta e il confronto. Non è un cast né sposta un verify nella base di E. Un field già confrontabile EUR non significa filtrare ordini di valuta nativa diversa.

```ml
type OccurrenceWhere<S> = Unfiltered | Where of OneOwnedMemberAtMost<
  CanonicalBool<2, ScalarPredicate<S> | OccurrenceTimePredicate<S> | OwnedMemberTest<S>>>

type OwnedMemberTest<S> = ExistsOwnedMember of {
  member : OwnedListFieldRef<S, M>;
  predicate : CanonicalBool<2, ScalarPredicate<M>>
}
```

Il test del membro è esistenziale sulla stessa riga, al massimo uno per occurrence where, e non può annidarne un altro. Sono ammessi i soli test temporali chiusi: membership in periodi/day/week/month dichiarati, fascia oraria locale, confronto di età rispetto a T e finestra rispetto a una data personale autorizzata. Non esistono regex, script, aggregazioni nel where o property-name lookup dinamici.

### Scope e tempo

```ml
type OccurrenceSource<U> =
  | Raw<RawBehaviorRef<U>> | SemanticRole<SemanticRoleRef<U>>
  | Repertoire<RepertoireRef<U>>
  | CertifiedOccurrence<CertifiedOccurrenceUseRef<U>>
  | CertifiedMilestone<CertifiedMilestoneUseRef<U>>

type BehaviorPopulation<U> = {
  source : OccurrenceSource<U>;
  grain : GrainProjectionRef;
  where : OccurrenceWhere
}
type FreeBehaviorScope<U> = { population : BehaviorPopulation<U>; period : ObservationPeriod }

type ObservationPeriod =
  | AbsoluteHalfOpen | ElapsedLookback | ElapsedAgoBand
  | ClosedCalendarRange | CompleteCalendarSeries
  | FullRelevantHistory<RelationshipHistoryRef>
  | AvailableHistory<DeclaredRetentionHistoryRef>
  | PersonalAnchored<PersonalWindow>

type CompleteCalendarSeries =
  | LastComplete<Day | Week | Month, PositiveInt>
  | ExplicitConsecutiveComplete<Day | Week | Month, CompleteCalendarBounds, PositiveInt>

type PersonalAnchor = CurrentAuthorizedDateTime
                    | CurrentAuthorizedDateAtDayStart<CalendarChoice>
type OccurrenceWindow<O> = BeforeSelfWithin<PositiveDuration>
                         | AfterSelfWithin<PositiveDuration> | AfterSelfUntilT

type CalendarChoice = FixedCalendar<CalendarDefinitionRef>
                    | CurrentProfileTimezone<AuthorizedTimezonePropertyRef, CalendarRulesRef>
```

AbsoluteHalfOpen porta start/end non ambigui e start<end; elapsed/ago portano durate e offset costanti; ClosedCalendarRange porta estremi di giorno/settimana/mese espliciti o relativi a T con posizioni costanti. CompleteCalendarSeries è finita, consecutiva, omogenea e non sovrapposta: mai una lista di bucket scelti per esito. PersonalWindow contiene solo intervalli prima/entro/dalla data personale, intervalli personali dichiarati e offset costanti ammessi; non timestamp expression. Una finestra di proprietà data può essere futura, per esempio rinnovo in `[T,T+Δ)`.

Source/repertoire, selector, grain projection, dedup identity, semantic identity e context restriction restano separati. Negli scope locali la key e l'ancora sono fornite dall'owner, non da lookup configurabili.

Bordi obbligatori: intervalli ordinari `[s,e)`; eventi usati sempre con timestamp<T; before/after occurrence tramite precedenza effettiva e ritardo<=Δ; before personal date `[a−Δ,a)`; entro Δ da data personale `[a,a+Δ]`; primi Δ del trial `[a,a+Δ)`; between steps strettamente nell'ordine; Human Response dalla prima apertura `[openedAt,T)` con lifecycle corretto. Deadline negativa inclusa matura solo con **T>d**, considerando gli eventi registrati fino a d inclusa e le eventuali garanzie contrattuali pertinenti, senza attestazioni esterne generali. Rolling burst è `[s,s+Δ)` contenuto nel periodo. Confronti di età preservano l'inclusività dell'operatore; `age>=7d` include il bordo esatto.

Calendario effettivo esplicito per rule; settimane lunedì–domenica; elapsed e calendar distinti. Date-only usano DayStart dichiarato; timezone mancante/ambigua non usa fallback. FullRelevantHistory comprende tutti gli eventi pertinenti registrati interrogabili fino a T, senza certificare l’inizio reale della relazione; AvailableHistory conserva l’intervallo esplicito dichiarato. Possono coincidere nei dati ma non sono alias automatici. Retention/cancellazione limitano il patrimonio e non generano da sole UNKNOWN. Nessuna variante o flag AST viene aggiunta.

Il `RelationshipHistoryRef` usato con `FullRelevantHistory` deve risolvere definizioni governate di fonte/repertorio compatibili con il dominio registrato. Una definizione 1.1 di storia della relazione reale richiede una nuova identità o `meaningRevision` immutabile, risolta e adottata esplicitamente: non basta cambiare etichetta o alias. `RegisteredQueryableHistory@1` è un riferimento illustrativo distinto, non la reinterpretazione della stessa identità/revisione precedente (§§3 e 5.1). Nessun nuovo costruttore AST o requisito di registry sulle condizioni Record.

## F3. Le cinque primitive e le loro union chiuse

### V

```ml
type ReadableAtom<C, T> =
  | CurrentProfile<CurrentProfileScalarRef<T>>
  | CertifiedCurrentScalar<CertifiedScalarUseRef<T, AuthorizedScalarSlot<C>>>
  | AuthorizedLocal<LocalScalarAccess<C, T>>
  | Reduced<AtomicReduction<AllowedLeafScope<C>, T>>

type AtomicOperand<C, T> = Read<ReadableAtom<C,T>> | LiteralParameter<Literal<T>>

type AtomicReduction<L, T> =
  | Count<L, OptionalSampleMinimum>
  | Sum<L, AdditiveQuantitativeField<T>, OptionalSampleMinimum>
  | Average<L, AverageableQuantitativeField<T>, PositiveInt>
  | Minimum<L, NumericOrTemporalField<T>, PositiveInt>
  | Maximum<L, NumericOrTemporalField<T>, PositiveInt>
  | CountDistinct<L, AuthorizedDistinctDimension, OptionalSampleMinimum>

type TypedComparison<C> =
  | PropertyTest<AuthorizedPropertyPredicate<C>>
  | Direct<ReadableAtom<C,A>, TypedComparisonOperator<A,B>, ReadableAtom<C,B> | Literal<B>>
  | Difference<ReadableAtom<C,A>, ReadableAtom<C,B>, LiteralTest<SignedDeltaLiteral<Unit>>>
  | Ratio<ReadableAtom<C,A>, ReadableAtom<C,B>, LiteralTest<DimensionlessLiteral>>
  | RelativeChange<ReadableAtom<C,A>, ReadableAtom<C,B>, LiteralTest<SignedFractionLiteral>>
  | FrequencyShare<OneWholeScope<C>, PartPredicate, PositiveInt, LiteralTest<Fraction0To1>>
  | WeightedShare<OneWholeScope<C>, PartPredicate, NonNegativeAdditiveWeight,
                  PositiveInt, OptionalMinimumWholeWeight, LiteralTest<Fraction0To1>>
  | UniqueWinnerTest<OneFrequencyDistribution<C>, EqualityOp, AuthorizedScalarValueTarget>
  | WinnerChangeTest<FixedDistributionPopulation<C>, PreviousPeriod, RecentPeriod,
                     MinimumForEach, CompareWinners | VerifyTwoLiteralValues>
  | MaximumValueShareTest<OneFrequencyDistribution<C>, Fraction0To1>
```

Tutti questi test concludono in Truth, mai in un operando. Ogni slot usa la firma locale autorizzata, non l'intera Rule. Le relazioni di compatibilità dei due input sono obbligatorie. Una distribution ha uno scope, una dimensione e un minimo>=1; WinnerChange usa la stessa popolazione/dimensione in due periodi. I winner restano interni. MaximumValueShare significa quota massima per valore<=soglia; non è uno scalar.

Count e distinct vuoti valgono 0; sum vuota vale 0 nella sua unità; average/min/max vuoti sono non definiti e il minimo obbligatorio fallisce. Ratio/relative/share richiedono denominatore positivo; share minimo>=1, stesso whole e peso quando applicabile; la part usa solo il budget residuo di owned-member predicate della stessa occurrence. Winner richiede unicità anche per “diverso”; parità certa fa fallire la guardia. Gli ignoti danno UNKNOWN solo se possono cambiare la verità del test completo, guardie incluse, non per la sola identità incerta del vincitore. Con Scienza=0, Arte=10, Tecnologia=10 e una occurrence di categoria ignota, UniqueWinner Equal Scienza è NO_MATCH; Different Scienza è UNKNOWN (06 §28.25). Campioni extra sono parametri espliciti, non condizioni sorelle da leggere. Non esistono formula, addizione, terminale riusabile o conversione implicita di unità.

### E

```ml
type SelectedExperience =
  | SingleExperience<SelectionBase, First | Last | Nth<1..10>, SingleExperienceBody>
  | LatestNPredicate<SelectionBase, N:1..10, SamePredicate, Some | All | CardinalityTest>
  | LastVersusPrevious<SelectionBase, OneField,
       SameValue | DifferentValue | NumericDifferenceTest | PositiveDenominatorRatioTest>
  | LastTwoIntervals<SelectionBase, RatioLiteralTest>
```

SelectionBase = source/grain/where pre-selettivo/periodo. SingleExperienceBody contiene solo proprietà/momento di Self, test tipizzati su scalari e scope foglia prima/dopo Self, outcome chiusi ed eventualmente **un** FirstFollowing. Il FirstFollowing sceglie il primo comportamento successivo fino a T e verifica solo proprietà/ritardo; non crea nuove finestre. Nessun E/K/P/T annidabile liberamente.

First lifetime = First sui pertinenti registrati in full relevant history, senza attestazione del passato reale; qualifying = where della base; Nth(1) normalizzabile a First. Latest N richiede N occurrence. LastPrevious legge lo stesso field sulle ultime due; LastTwoIntervals usa ultime tre e due durate positive native. Nessuna posizione/campo arbitrario, tie-break tecnico o fallback verso occurrence che soddisfano meglio verify.

### K

```ml
type IdentitySet =
  | ObservedValues<ObservedBase, ObservedBody, CandidateQuantifier>
  | CorrelatedEventObjects<EventIdentityBase, PrimaryAccess, CorrelatedBody, CandidateQuantifier>
  | Opportunities<OpportunityBase, AllInBase | LatestN<1..10>, OpportunityBody, CandidateQuantifier>
  | AuthoritativeObjects<AuthoritativeObjectCondition>

type CandidateQuantifier<G> =
  | Some | None
  | All of { minimumCandidates : PositiveInt }
  | Cardinality of CardinalityTest
  | Share of { minimumCandidates : PositiveInt; test : LiteralTest<Fraction0To1> }

type CardinalityTest = AtLeast<Nat> | AtMost<Nat> | Exactly<Nat>
                    | BetweenInclusive<lower:Nat, upper:Nat>
```

ObservedBase introduce valori distinti di una dimensione del medesimo repertorio registrato. ObservedBody consente quel valore, misure foglia sullo stesso repertorio e valore, e presenza/assenza fra le occurrence interrogabili anteriori all’inizio del periodo recente, senza un inizio della relazione reale certificato. Count distinct order/day è una riduzione, non un secondo insieme. Nessuna correlazione cross-repertoire per semplice uguaglianza di valori.

EventIdentityBase introduce una semantic identity unica dal seme. CorrelatedBody ha fino a quattro scope tutti same-key; proprietà primarie sono disponibili solo con UniquePrimary certificato oppure pivot First/Last della sorgente primaria per quella identity. Finestre indipendenti o del pivot esplicite. Non cambia key, non naviga altri oggetti.

OpportunityBase identifica prima key e inizio pertinente: prima consegna registrata per le origini event-based, con i requisiti specifici di lifecycle per le altre origini; FirstMessageDelivery, FirstCampaignDeliveryToProfile, InitialTicketOpening o ApprovedUniqueDomainStart. Solo dopo applica startPeriod/qualificatori, eventuale explicit mature-only e latest N; infine il body. OpportunityBody contiene proprietà dell'opportunità e outcome attribuiti alla stessa opportunità, entro Δ oppure dopo start fino a T. Non apre altre opportunità. Mature-only non è implicito.

AuthoritativeObjectCondition è una union distinta di StateOnlyObjects, NamedEvidenceObjects, CurrentCarts e CurrentDesignatedTrial, con body delle sole rispettive firme di F5. I candidati precedono il body; UNKNOWN non scompare dalla base o dal denominatore. All è non vacuo; None su vuoto è MATCH; share di candidati conta candidati, non occurrence di esito. Non esiste `V.Share(KQuery)`.

K quantifica composizionalmente i body individuali: con base determinata D=S+F+U, l’inviluppo è [S,S+U], non necessariamente l’insieme esatto dei conteggi globalmente realizzabili. Some è MATCH per S≥1, NO_MATCH per S+U=0, altrimenti UNKNOWN; None inverte quei due esiti certi. Cardinality usa l’inviluppo; All dà NO_MATCH per minimo insoddisfatto o F≥1, MATCH con guardia soddisfatta e U=0, altrimenti UNKNOWN; Share usa [S/D,(S+U)/D] con D e guardie conservati. Non si restringono i limiti con inferenze globali fra candidati distinti. Due ordini certi e un solo Refund con orderId ignoto ma univalore danno Some “ordine senza rimborsi” UNKNOWN: una prova globale sarebbe possibile, ma non appartiene all’algebra approvata (07 §9.1; 06 §§12.6, 28.24). È la scelta N2 della versione 1.2 approvata esplicitamente, non un’opzione degli executor. E conserva invece le selezioni compatibili, P la chiave comune all’intero blocco, share part/whole condivisi, guardie e ownership.

### P

```ml
type PathStep = { base : FreeBehaviorScope<PathStepUse>;
                  select : AnyOccurrence | FirstInBase | LastInBase }

type PathLayout = Independent<PlainChain<2..5>> | Correlated<exists K. CorrelatedLayout<K>>
type CorrelatedLayout<K> =
  | Whole<KeyedChain<K,2..5>>
  | Starts<KeyedChain<K,B>, PlainBridge, PlainChain<S>>
  | Ends<PlainChain<P>, PlainBridge, KeyedChain<K,B>>
  | Middle<PlainChain<P>, PlainBridge, KeyedChain<K,B>, PlainBridge, PlainChain<S>>
(* B>=2; P/S presenti>=1; somma di tutti i passi<=5. *)

type OrderedPath = {
  layout : PathLayout;
  adjacentDurations : OptionalClosedDurationConstraintForEachAdjacentEdge;
  totalDuration : OptionalClosedDurationConstraint;
  negativeChecks : OptionalBudgetedBool2<AuthorizedBetweenAbsenceLeaf>;
  finalOutcome : Optional<AuthorizedProfileOrSameKeyOutcomeFromFinal>
}
```

C'è una sola key dichiarata nel KeyedChain; prefix e suffix non la ereditano. Le basi dei passi rimangono profile-scoped: First/Last si applica prima di cercare tuple e prima della correlazione, non per key. Servono occurrence distinte, precedenza effettiva e un valore comune all'intero blocco per key multivalore. Nessuna scelta greedy per AnyOccurrence.

Le assenze possono riferirsi solo ad adiacenti o primo–ultimo, usando handle locali di slot dichiarati dal layout. Un handle keyed esiste solo per archi interni al blocco e per primo–ultimo nel caso Whole. Il finale può leggere la key solo in Whole/Ends, mai in Starts/Middle. L'esito finale è presenza/assenza entro Δ o fino a T, non un sotto-percorso. Negative e finale consumano il budget di quattro; i passi propri non lo consumano.

### T

```ml
type ActivityPattern =
  | DistinctActiveDays<Scope, CardinalityTest>
  | CalendarDayBurst<Scope, MinimumOccurrencesInDay>
  | RollingWindowBurst<Scope, PositiveDuration, MinimumOccurrences>
  | Streak<Scope, MinimumConsecutiveDays>
  | MaximumGap<Scope, MaximumElapsedGap>
  | ResumedAfterInactivity<CanonicalReturnDefinition, ReturnObservationScope>
  | RepeatedReturns<CanonicalReturnDefinition, ReturnObservationScope, CardinalityTest>
  | LastQualifiedReturn<CanonicalReturnDefinition, SelectionPeriod, SingleExperienceBody>
  | QualifiedPeriodCardinality<Population, CompleteCalendarSeries,
                                ClosedPerPeriodRecognizer, CardinalityTest>

type ClosedPerPeriodRecognizer =
  | ActivePeriod | AtLeastDistinctActiveDays<PositiveInt>
  | CalendarDayBurstAtLeast<PositiveInt>
  | RollingBurstAtLeast<PositiveDuration, PositiveInt>
  | ConsecutiveActiveDaysAtLeast<PositiveInt>
  | MaximumGapAtMost<PositiveDuration>
```

EveryPeriod, NOfMPeriods e NDaysPerPeriod sono forme chiuse normalizzate a QPC con ActivePeriod/DistinctDays e cardinalità corrispondente. QPC conserva M periodi completi, consecutivi e fissi; il recognizer usa lo stesso source/grain/where entro ogni bucket. Non contiene Bool, Rule, QPC, formule o output riusabili. AtLeast N>M è invalido. I vuoti/UNKNOWN non vengono eliminati; zero eventi registrati rende il bucket vuoto, mentre dati esistenti decisivi possono renderlo UNKNOWN; count qualificanti interno in [S,S+U]. MaxGap su vuoto vale la durata dell'intero periodo, non fallisce automaticamente per un campione minimo aggiunto.

MaximumGap richiede entrambi gli estremi del periodo finiti e determinati dalla forma: FullRelevantHistory da solo è invalido; AvailableHistory è ammesso con un intervallo dichiarato completo di estremi, come gli altri periodi delimitati autorizzati. Non si sostituisce implicitamente l’inizio con il primo evento o un limite di retention. Estremo strutturalmente assente→configurazione invalida; estremo previsto di valore ignoto sul profilo→possibile UNKNOWN quando decisivo. Il ciclo corrente vuoto all’inizio resta ammesso con gap=0. Per QPC valgono i bordi finiti dei bucket; nessun ulteriore divieto per burst/streak su FullRelevantHistory (07 §13.3; 06 §§14.4, 28.23).

CanonicalReturnDefinition è FromActivity(repertorio/predicati fissi, pausa minima) oppure una CertifiedReturnUseRef. In FromActivity, u≺r sono entrambi registrati e consecutivi, pausa>=Δ e nessun intermedio registrato: non serve un certificato esterno. Il primo registrato senza predecessore non è un ritorno; retention non inventa u. Un episodio per r. I fatti certificati conservano il proprio contratto. ReturnTimeIn(W) e EntirePauseIn(W) restano distinti; il secondo richiede u/r/pausa in W. LastQualifiedReturn seleziona prima di verify con il body tecnico E senza annidare E. Increase/decrease sono V; stopped è macro; ultimi-due-intervalli è E.

## F4. Context/scoping rules e limiti

Il contesto root contiene soltanto CurrentProfile, T, calendario e definizioni. Ogni E/K/P/T contestuale introduce un token owner fresco, nascosto alla chiusura della rule:

```text
Γ ; O:Role ; AuthorizedSlots(Role) ⊢ body : Truth
─────────────────────────────────────────────
Γ ⊢ CloseOwner(body) : Rule
```

LocalScalar<Root> è Never. I riferimenti sono normalmente Self, key del candidato, pivot o slot implicito. Solo P richiede, quando necessario, handle di intervallo finiti e owner-relative; non ID utente di occurrence. Gli handle non sono valori temporali e non compaiono negli input delle sibling.

Le leaf union sono chiuse per owner: Selected; ObservedValue; CorrelatedObject; Opportunity; StateOnly; NamedEvidenceObject; Cart; CartProduct; Trial; Path; QualifiedReturn. Non esiste una union dei loro slot utilizzabile come universal body. I test V locali ricevono soltanto field, riduzioni, dimensioni e periodi della rispettiva firma; una capability di registry seleziona tag esistenti, non crea espressioni nuove.

Le famiglie `AllowedLeafScope<C>` sono chiuse: Root→Free; Selected/Return→RelativeToSelf; Observed→SameRepertoireAndDimension; Correlated→SameIdentity; NamedObject→NamedScope dello stesso contratto; CartProduct→SameProductBehavior; Trial→AttributedUsage oppure UpgradeTemporal. Opportunity/Path/StateOnly/Cart root non ricevono un V comportamentale generico: hanno le rispettive foglie specifiche. Local scalars sono solo i fatti Self/candidato/primaria/pivot/lifecycle/membro dichiarati dalla riga; Root non ne ha. Le distribuzioni locali, dove autorizzate come test tipizzati, non esportano dimensioni/vincitori; Observed può usare come dimensione di distribuzione solo la D già vincolata. CountDistinct conserva i propri slot dichiarati Order/CalendarDay senza creare insiemi navigabili. Nei contratti ogni operazione richiede la capability specifica, non la sola presenza di field compatibili.

Limiti principali preservati nella versione 1.2: massimo quattro livelli di gruppi AND/OR, radice inclusa, e 100 nodi discendenti, radice esclusa. Limiti interni: body/where Bool 2; scope subordinati 4 per owner; nth/latest N/latest N opportunities 1–10; P 2–5 passi; un blocco di almeno due passi e una key; un member frame solo Cart→Product; un FirstFollowing nel body che lo autorizza; una verifica owned-list nel singolo occurrence where.

I costruttori hanno indici finiti di profondità/budget o wrapper nominali controllati. Due usi di scope costano due posti, anche con hash identico. Una guardia sullo stesso scope e la part interna di una share non creano scope. Il Cart include nel proprio budget le foglie del membro. Nessuna deduplication ripara un superamento originario dei limiti.

## F5. Registry contracts

**Oggetto autorevole:** definizione versionata con identity/lifecycle, appartenenza al profilo, inventario, autorità/riconciliazione, validità corrente, proprietà e lifecycle fields autorizzati. Shape chiuse:

| Shape | Body/slot consentiti |
|---|---|
| StateOnly | Proprietà correnti dichiarate; baseline Subscription |
| NamedEvidence | Proprietà/lifecycle più named scopes del medesimo owner; baseline Ticket/Human Response, eventuale Order/Refund approvato |
| Cart | Stato/ultimo aggiornamento/completamento correnti; un solo passaggio ai propri prodotti correnti; behavior del profilo sullo stesso prodotto |
| DesignatedTrial | Trial corrente univocamente designato; Usage attribuito al trial; solo Upgrade del profilo dall'inizio a T come slot temporale distinto |

**Named evidence:** nome/versione, owner contract, business meaning, grain, typed properties, action timestamp/order, orizzonti ammessi, appartenenza certificata alla stessa object identity, completezza e tag di operazioni foglia autorizzate. Non è RelationTraversal. Ticket openedAt significa apertura iniziale del lifecycle; una riapertura non la sostituisce. Trial Upgrade non è un named scope correlato e non richiede trial ID. Il subject è il profilo cui l'evidenza è attribuita, non necessariamente l'attore materiale di una risposta.

**Certified evidence:** union Scalar / Occurrence / Milestone / CurrentState / CanonicalReturnEpisode. Ogni definition dichiara, quando pertinenti, semantic value/type, identity, grain, timestamp e ordine, validità, popolazione/periodo/soglia, unità, observation/completeness e consumer autorizzati. Historical property è un field certificato della relativa occurrence/milestone, non un lookup.

I reference di uso sono distinti per forma e consumer. Scalar non può essere OccurrenceSource. CurrentState è consumato via property slot o approved object backing. Milestone richiede la qualità di attainment/first-attainment effettivamente consumata. ReturnEpisode serve soltanto Resumed/Repeated/LastQualifiedReturn. I parametri della definition sono un prodotto finito di letterali, reference semantici e time specifications chiuse, mai risultati di rule o producer invocation. L'AST legge la definizione, non l'algoritmo.

Per S012 devono essere certificati primo raggiungimento del 90% cumulato dall'inizio di W, content identity/version/durata pertinente, identità episodio/serie, tempo e disponibilità storica del successivo episodio alla seconda milestone. Non esistono slot video nell'AST e il producer non è obbligatorio nel core.

**Validazione contestuale aggiuntiva:** ammissibilità Behavioral solo in Profiles API/Admin ed export dei profili verso app/file/database; nelle altre pipeline è disponibile soltanto Record, dove già consentito. Le forme nuove fuori contesto sono rifiutate dal server; non producono un risultato ternario.

## F6. Validation layers e evidence requirements

```ml
type StaticAssessment =
  | InvalidShape
  | InvalidAgainstRegistry
  | UnresolvedSemanticReference
  | ValidButUnavailableInWorkspace
  | ReadyForEvaluation
```

Shape/checked values impongono tag, profondità, arità, budget, posizioni, periodi, soglie, contiguità/blocchi e divieti. **RecordAdmissibility conserva schema/path, operatore, tipo e operandi esistenti, senza nuovi registry o meaningRevision.** Per i nuovi predicati governati il registry verifica domini/unità/grana, ruoli, owner/slot, operatori/riduzioni e consumer. I normali eventi non richiedono certificati di completezza esterna o inizio reale della relazione; inventari, lifecycle, named scopes e CDE conservano le proprie garanzie. Definizione governata irrisolta e adapter assente rimangono stati distinti.

Ogni combinazione fonte/schema/predicato ammessa in Behavioral conserva significato e requisiti informativi indipendenti dall’executor. Dati esistenti decisivi ignoti o garanzie contrattuali insufficienti seguono le regole sull’evidenza. L’assenza di occurrence registrate, anche per retention o senza utente di sorgente, non è automaticamente UNKNOWN. Repertorio configurato vuoto è valutabile; capability strutturale assente è non disponibile. Nessun registry retroattivo per Record né correzioni retroattive agli executor.

La valutazione richiede dati esistenti adeguati su valori, key/dedup, timestamp, ordine e maturazione; inventory, lifecycle, validità, storia e limiti certificati sono richiesti dove il contratto li prevede. Non si aggiungono occurrence esterne ipotetiche. Count/assenza su base registrata vuota danno 0/MATCH; l’esistenza è NO_MATCH. Un Purchase con NativeCurrency ignota rimane possibile Purchase EUR. Minimum/positive denominator/unique winner/required N/non-empty restano guardie chiuse: falsa certa→NO_MATCH; incerta→UNKNOWN quando decisiva.

AND è NO_MATCH se almeno un figlio è NO_MATCH, MATCH se tutti sono MATCH, UNKNOWN altrimenti. OR è MATCH se almeno un figlio è MATCH, NO_MATCH se tutti sono NO_MATCH, UNKNOWN altrimenti. Failure tecnico resta distinto; nessun includeUnknown o last-known-match.

La Fase 4A non attesta la disponibilità concreta di contratti o producer, né l'esecuzione degli acceptance test. Il catalogo reale dei **26 operatori è noto** e il raccordo concettuale `FilterCondition → V.RecordCondition` conserva gli operatori e le forme pubbliche, con validazione record esistente e senza dipendenze aggiuntive dal registro governato. Restano aperti il formato/persistenza delle nuove `FilterRule` e la verifica di migrazione dei filtri installati; non si introduce un `legacyMode` semantico.

**Separazione pubblica/interna:** ordinamento, flattening, dedup ed eliminazione dei singleton avvengono soltanto nel Canonical AST interno dopo validazione. Non modificano implicitamente la sintassi pubblica, i filtri esistenti persistiti, `Where.Equal` o i fingerprint dell’Admin.

Per la progettazione si assume una vista coerente di profili/associazioni/evidenze e un matching valido fornito dalla destination; **non si dichiara che le garanzie siano implementate**. IR/pubblicazione, acquisizione/attribuzione KPID e matching sono tre lavori rinviati, non risolti: non bloccano il progetto del compilatore/piano fisico e possono condizionare il rilascio. La latenza tecnica non diventa automaticamente UNKNOWN; lettura fallita/parziale produce failure. Non si adottano campi ReadView, generazioni, watermark, tagli di ingestione o materializzazione del rapporto 12. Membership rimane distinta da consensi/matching/modalità di export.

## F7. Canonicalization e provenance

Prima ammissione nella grammatica, poi normalizzazione: niente recupero di empty group, slot illeciti o budget eccessivi tramite cancellazione. Eliminare singleton, flatten stesso operatore, ordinare e deduplicare figli booleani; literal esatti secondo il catalogo; reference su significato immutabile; owner modulo rinomina strutturale; espansione dei default semantici. K AtLeast(1)→Some, Exactly(0)→None, Between(n,n)→Exactly(n); preservare minimi di All/share. Nth(1)→First. Le tre forme calendar Every/NofM/NDaysEach normalizzano a QPC equivalente.

Non spostare prefiltri/verify, scope fuori dall'owner, recent/baseline o key fuori dal blocco. Non fondere ratio/share, current/historical, correlated/independent, calendar/elapsed, available/full history, deadline/UntilT. Non sostituire null con zero, ultimo con ultimo non-null o stato autorevole con ultimo evento. Non semplificare x=x a true su valori ignoti. Equality canonica è strutturale normalizzata, non equivalenza logica completa; l'hash identifica la richiesta, non da solo un risultato riusabile a qualunque T.

Recipe identity/version, display intent, original parameters, customized-from e grouping hints sono metadata separati, collegati a indirizzi semantici solo per presentazione. Rimuoverli non cambia il risultato. Tutte le definizioni, guardie e mapping necessari sono nell'AST o nei registry effettivamente referenziati.

## F8. Extension points e confine di evoluzione

Nuovi scalar predicate, atomic reductions, preset E, T recognizer e nuove forme CDE richiedono un'estensione versionata delle rispettive union chiuse. Un nuovo recognizer T non entra automaticamente nel sotto-catalogo QPC. Nuovi mapping, named evidence scopes, producer per forme esistenti e contratti approvati nelle shape esistenti sono registry extensions. Un nuovo tipo di slot/body richiede una nuova shape chiusa. Aumenti dei limiti possono essere limit-catalog/configuration extensions approvate con aggiornamento dei bounded types, senza autorizzare nuovi ruoli o combinazioni e senza fingere che la v1.0 avesse già limiti differenti.

Sono invarianti anche nelle estensioni: nessuna formula generale, arbitrary join, sibling variable, relation graph, query ricorsiva, ranking dinamico, nth-to-nth libero, statistica/time series generica, storico universale, cross-profile correlation, NOT o workflow. Una pubblicazione esterna certificata può rendere leggibile un risultato, non rendere inline il relativo algoritmo.

I094, I097/S015 e la ricostruzione I098 da cronologia separata restano deferred; la lettura di uno snapshot storico già certificato non introduce il relativo producer.

**Fine del Canonical AST — Phase 4A Final Model.**
