# Canonical AST — Phase 4A Final Model

**Documento estraibile — revisione proposta 1.1.** Base normativa: *Behavioral Filtering Product & Semantic Specification — proposta riconciliata v1.1*. Il documento 06 v1.0 resta storico finché la revisione non è approvata. Questa sezione definisce il modello tecnico canonico da usare come input della Fase 4B. La notazione è ML-like, non JSON; i nomi non sono field pubblici. Non definisce API, serialization schema, migration protocol o execution plan.

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

AllProfiles rappresenta il **filtro pubblico assente**, non un gruppo vuoto. La radice pubblica, quando esiste, è AND/OR; empty group è invalido; singleton e gruppi omogenei consecutivi sono validi nel documento pubblico. **Massimo quattro gruppi principali e 100 discendenti** (root esclusa). La grammatica **interna** può eliminare singleton e appiattire gli operatori uguali, ma non modifica il documento pubblico, la sua persistenza o l’uguaglianza esistente. Non esiste NOT generico.

**Raccordo del riepilogo:** le cinque varianti top-level appartengono all’AST di Unified; il DTO pubblico rimane `core.Filter` e `FilterCondition` per le condizioni Record. Le 26 condizioni sono property test V e foglie ammesse nei body solo con le relative capability. La radice pubblica resta un gruppo AND/OR, anche se il canonical AST può usare una foglia dopo normalizzazione.

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

Esiste un percorso raw tramite schema tipizzato/versionato, senza ontology obbligatoria. Il raw schema non certifica da solo un ruolo business. Product ID e Order ID, namespace diversi, EUR/USD e quantità di dimensioni diverse non sono implicitamente compatibili. I count con grane diverse possono essere confrontati e usati in una ratio, mantenendo entrambe le grane. Sono necessari witness statici di compatibilità specifici dell'operazione; non un cast numerico generale.

### Predicate chiusi

`ScalarPredicate<S>` comprende `RecordCondition` (i **26 operatori del record filtering** con tipi, payload e validazione di `d69e6168`) e i nuovi predicati governati chiusi: equality/difference con literal o current-profile scalar compatibile; confronto ordinato/intervallo; membership; missing/present/explicit null/known; empty text; contains/starts/ends; appartenenza nei set; test monetario e temporale. Il record filtering conserva anche String ordinabile, Time/Year, JSON dinamico, contenitori e negativi autonomi; i nuovi predicati non ne restringono il catalogo. Nessun NOT generico né conversione implicita dei literal in property reference.

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

Bordi obbligatori: intervalli ordinari `[s,e)`; eventi usati sempre con timestamp<T; before/after occurrence tramite precedenza effettiva e ritardo<=Δ; before personal date `[a−Δ,a)`; entro Δ da data personale `[a,a+Δ]`; primi Δ del trial `[a,a+Δ)`; between steps strettamente nell'ordine; Human Response dalla prima apertura `[openedAt,T)` con lifecycle corretto. Deadline negativa inclusa matura solo con **T>d** e copertura fino a d inclusa. Rolling burst è `[s,s+Δ)` contenuto nel periodo. Confronti di età preservano l'inclusività dell'operatore; `age>=7d` include il bordo esatto.

Calendario effettivo esplicito per rule; settimane baseline lunedì–domenica; giorni elapsed e calendar non sono intercambiabili. Date-only come ancore usano DayStart dichiarato. Timezone mancante/ambigua non usa fallback nascosti. FullRelevantHistory e AvailableHistory non sono sinonimi; il requisito di storia dipende dall'uso, quindi Last non richiede automaticamente il prefisso necessario a First lifetime.

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

Count e distinct vuoti valgono 0; sum vuota vale 0 nella sua unità; average/min/max vuoti sono non definiti e il minimo obbligatorio fallisce. Ratio/relative/share richiedono denominatore positivo; share minimo>=1, stesso whole e peso quando applicabile; la part usa solo il budget residuo di owned-member predicate della stessa occurrence. Winner richiede unicità anche per “diverso”; parità certa fa fallire la guardia. Campioni extra sono parametri espliciti, non condizioni sorelle da leggere. Non esistono formula, addizione, terminale riusabile o conversione implicita di unità.

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

First lifetime = First su full relevant history; qualifying = where della base; Nth(1) normalizzabile a First. Latest N richiede N occurrence. LastPrevious legge lo stesso field sulle ultime due; LastTwoIntervals usa ultime tre e due durate positive native. Nessuna posizione/campo arbitrario, tie-break tecnico o fallback verso occurrence che soddisfano meglio verify.

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

ObservedBase introduce valori distinti di una dimensione del medesimo repertorio. ObservedBody consente quel valore, misure foglia sullo stesso repertorio e valore, e presenza/assenza nella storia anteriore all'inizio del periodo recente. Count distinct order/day è una riduzione, non un secondo insieme. Nessuna correlazione cross-repertoire per semplice uguaglianza di valori.

EventIdentityBase introduce una semantic identity unica dal seme. CorrelatedBody ha fino a quattro scope tutti same-key; proprietà primarie sono disponibili solo con UniquePrimary certificato oppure pivot First/Last della sorgente primaria per quella identity. Finestre indipendenti o del pivot esplicite. Non cambia key, non naviga altri oggetti.

OpportunityBase identifica prima key e inizio storico: FirstMessageDelivery, FirstCampaignDeliveryToProfile, InitialTicketOpening o ApprovedUniqueDomainStart. Solo dopo applica startPeriod/qualificatori, eventuale explicit mature-only e latest N; infine il body. OpportunityBody contiene proprietà dell'opportunità e outcome attribuiti alla stessa opportunità, entro Δ oppure dopo start fino a T. Non apre altre opportunità. Mature-only non è implicito.

AuthoritativeObjectCondition è una union distinta di StateOnlyObjects, NamedEvidenceObjects, CurrentCarts e CurrentDesignatedTrial, con body delle sole rispettive firme di F5. I candidati precedono il body; UNKNOWN non scompare dalla base o dal denominatore. All è non vacuo; None su vuoto è MATCH; share di candidati conta candidati, non occurrence di esito. Non esiste `V.Share(KQuery)`.

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

EveryPeriod, NOfMPeriods e NDaysPerPeriod sono forme chiuse normalizzate a QPC con ActivePeriod/DistinctDays e cardinalità corrispondente. QPC conserva M periodi completi, consecutivi e fissi; il recognizer usa lo stesso source/grain/where entro ogni bucket. Non contiene Bool, Rule, QPC, formule o output riusabili. AtLeast N>M è invalido. I vuoti/UNKNOWN non vengono eliminati; count qualificanti interno in [S,S+U]. MaxGap su vuoto vale la durata dell'intero periodo, non fallisce automaticamente per un campione minimo aggiunto.

CanonicalReturnDefinition è FromActivity(repertorio/predicati fissi, pausa minima) oppure una CertifiedReturnUseRef equivalente. Ritorno = u≺r consecutivi, pausa>=Δ osservata, predecessore reale; un episodio per r. ReturnObservationScope è ReturnTimeIn(W) o EntirePauseIn(W); il secondo impone u/r/pausa dentro W. LastQualifiedReturn seleziona l'ultimo ritorno qualificante prima di verify, riusando il body tecnico di E senza annidare una E generica. Increase/decrease sono V; stopped è macro; ultimi-due-intervalli è E.

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

Limiti proposti 1.1: root Bool **4 gruppi / 100 nodi discendenti pubblici**; body/where Bool 2; scope subordinati 4 per owner; nth/latest N/latest N opportunities 1–10; P 2–5 passi; un blocco di almeno due passi e una key; un member frame solo Cart→Product; un FirstFollowing nel body che lo autorizza; una verifica owned-list nel singolo occurrence where.

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

**Validazione contestuale aggiuntiva:** ammissibilità Unified solo in Profiles API/Admin ed export dei profili verso app/file/database; nelle altre pipeline è disponibile soltanto Record, dove già consentito. Le forme nuove fuori contesto sono rifiutate dal server; non producono un risultato ternario.

## F6. Validation layers e evidence requirements

```ml
type StaticAssessment =
  | InvalidShape
  | InvalidAgainstRegistry
  | UnresolvedSemanticReference
  | ValidButUnavailableInWorkspace
  | ReadyForEvaluation
```

Shape/checked values impongono tag, profondità, arità, budget, posizioni, limiti, periodi e soglie valide, contiguità/numero blocchi e divieti di nesting. Il registry verifica schema, domini/unità/grana, ruoli, owner/slot, permitted operators/reducers, consumer dell'evidenza e requisiti strutturali di osservabilità/storia. Una definizione sconosciuta non permette di attestare validità; una definizione valida senza adapter di workspace è invece una capability indisponibile.

La valutazione per profilo richiede evidenza su valori, key/dedup, proprietà, inventory, ordine, storia, maturazione, validità e limiti quantitativi pertinenti. Dati incerti possono produrre UNKNOWN, salvo prove decisive. Minimum/positive denominator/unique winner/required N/non-empty sono guardie chiuse: falsa certa→NO_MATCH; incerta→UNKNOWN quando decisiva. Non vengono configurate come espressioni arbitrarie.

AND è NO_MATCH se almeno un figlio è NO_MATCH, MATCH se tutti sono MATCH, UNKNOWN altrimenti. OR è MATCH se almeno un figlio è MATCH, NO_MATCH se tutti sono NO_MATCH, UNKNOWN altrimenti. Failure tecnico resta distinto; nessun includeUnknown o last-known-match.

La Fase 4A non attesta la disponibilità concreta di contratti o producer, né l'esecuzione degli acceptance test. Il catalogo reale dei **26 operatori è noto** e il raccordo concettuale `FilterCondition → V.RecordCondition` conserva gli operatori e le forme pubbliche. Restano aperti il formato/persistenza delle nuove `FilterRule` e la verifica di migrazione dei filtri installati; non si introduce un `legacyMode` semantico.

**Separazione pubblica/interna:** ordinamento, flattening, dedup ed eliminazione dei singleton avvengono soltanto nel Canonical AST interno dopo validazione. Non modificano implicitamente la sintassi pubblica, i filtri esistenti persistiti, `Where.Equal` o i fingerprint dell’Admin.

## F7. Canonicalization e provenance

Prima ammissione nella grammatica, poi normalizzazione: niente recupero di empty group, slot illeciti o budget eccessivi tramite cancellazione. Eliminare singleton, flatten stesso operatore, ordinare e deduplicare figli booleani; literal esatti secondo il catalogo; reference su significato immutabile; owner modulo rinomina strutturale; espansione dei default semantici. K AtLeast(1)→Some, Exactly(0)→None, Between(n,n)→Exactly(n); preservare minimi di All/share. Nth(1)→First. Le tre forme calendar Every/NofM/NDaysEach normalizzano a QPC equivalente.

Non spostare prefiltri/verify, scope fuori dall'owner, recent/baseline o key fuori dal blocco. Non fondere ratio/share, current/historical, correlated/independent, calendar/elapsed, available/full history, deadline/UntilT. Non sostituire null con zero, ultimo con ultimo non-null o stato autorevole con ultimo evento. Non semplificare x=x a true su valori ignoti. Equality canonica è strutturale normalizzata, non equivalenza logica completa; l'hash identifica la richiesta, non da solo un risultato riusabile a qualunque T.

Recipe identity/version, display intent, original parameters, customized-from e grouping hints sono metadata separati, collegati a indirizzi semantici solo per presentazione. Rimuoverli non cambia il risultato. Tutte le definizioni, guardie e mapping necessari sono nell'AST o nei registry effettivamente referenziati.

## F8. Extension points e confine di evoluzione

Nuovi scalar predicate, atomic reductions, preset E, T recognizer e nuove forme CDE richiedono un'estensione versionata delle rispettive union chiuse. Un nuovo recognizer T non entra automaticamente nel sotto-catalogo QPC. Nuovi mapping, named evidence scopes, producer per forme esistenti e contratti approvati nelle shape esistenti sono registry extensions. Un nuovo tipo di slot/body richiede una nuova shape chiusa. Aumenti dei limiti possono essere limit-catalog/configuration extensions approvate con aggiornamento dei bounded types, senza autorizzare nuovi ruoli o combinazioni e senza fingere che la v1.0 avesse già limiti differenti.

Sono invarianti anche nelle estensioni: nessuna formula generale, arbitrary join, sibling variable, relation graph, query ricorsiva, ranking dinamico, nth-to-nth libero, statistica/time series generica, storico universale, cross-profile correlation, NOT o workflow. Una pubblicazione esterna certificata può rendere leggibile un risultato, non rendere inline il relativo algoritmo.

I094, I097/S015 e la ricostruzione I098 da cronologia separata restano deferred; la lettura di uno snapshot storico già certificato non introduce il relativo producer.

**Fine del Canonical AST — Phase 4A Final Model.**
