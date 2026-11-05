# Specifica del catalogo eventi CDP — consegna per Codex

**Ambito:** soltanto definizioni degli eventi, corrispondenza con i messaggi ricevuti, tassonomia, preset, varianti, semantica, gestione e validazione delle definizioni. **Non contiene istruzioni su come sviluppare MCP, tool, API, trasporto, database, query SQL o interfacce di esecuzione.**

| Metadato | Valore |
|---|---|
| Base dati | `cdp-event-taxonomy.zip`, pacchetto versione **1.1.0** |
| Fonte primaria | `cdp-event-catalog.json` |
| Varianti separate | `cdp-event-bindings.json` |
| Verifica documentale del catalogo | **29 ottobre 2026** |
| Lingua dei messaggi tecnici | Inglese, preservando i nomi canonici |
| Lingua di etichette, descrizioni, sinonimi | Italiano (`it-IT`) |
| Dimensione | **102 definizioni** e **16 varianti tecniche** separate |

## 1. Obiettivo e confini

Il CDP raccoglie messaggi secondo un modello compatibile con Segment e RudderStack e associa tali messaggi ai profili. L'amministratore deve poter definire e personalizzare un catalogo di **eventi semantici**; chi costruisce un filtro deve poter selezionare un evento leggibile senza dover impostare manualmente `type`, `event` e `name`.

Un evento del catalogo è un **concetto stabile** (per esempio `commerce.order_completed`, «Acquisto completato»), distinto dal **messaggio tecnico** che lo rappresenta (per esempio `{ "type": "track", "event": "Order Completed" }`). Il testo italiano aiuta a riconoscere il significato; **solo i matcher tecnici** determinano se un messaggio concreto appartiene alla definizione.

La fonte dei nomi tecnici, degli identificatori, dei matcher, delle descrizioni, dei sinonimi e dei riferimenti documentali è il **JSON effettivamente consegnato**, non un nome inventato, una traduzione o una deduzione da esempi narrativi. L'inventario nell'appendice di questo documento riproduce integralmente i 102 eventi per gli aspetti essenziali; per tutti i 20 campi va usato il JSON.

**Fuori ambito di questo documento:** implementazione del protocollo MCP, contratti o nomi dei tool, sintassi della DSL dei filtri, algoritmi di segmentazione, modello di database e politiche generali di identità. Queste scelte appartengono ad altre specifiche. Le osservazioni su conteggi, assenze e identità riportate qui servono soltanto a evitare un'interpretazione errata degli eventi.

## 2. Artefatti di riferimento e precedenza

I seguenti file del pacchetto devono essere disponibili insieme a questa specifica:

| File | Ruolo nel lavoro sugli eventi |
|---|---|
| `cdp-event-catalog.json` | **Autorevole** per le 102 definizioni, i 20 campi, i nomi, i matcher e i preset |
| `cdp-event-catalog.schema.json` | Validazione strutturale del catalogo |
| `cdp-event-bindings.json` | **Autorevole** per le 16 varianti e i loro stati iniziali |
| `cdp-event-bindings.schema.json` | Validazione strutturale del registro delle varianti |
| `fixtures/expected-catalog.json` | Inventario atteso e controlli di completezza |
| `fixtures/matcher-cases.json` | Casi positivi/negativi di riconoscimento degli eventi |
| `README.md` | Istruzioni di interpretazione e validazione del pacchetto |
| `validate_catalog.py` e `requirements.txt` | Validazione riproducibile dei file originari; contiene anche test di altri ambiti, da non confondere con requisiti implementativi sugli eventi |

Per questo incarico **non occorre modificare o implementare** `cdp-mcp-examples.json` e `cdp-filter-dsl.schema.json`.

In caso di contrasto:

1. Per **ID, `type`/`event`/`name`, matcher, descrizioni, sinonimi, fonte, appartenenza e flag del preset**, prevale `cdp-event-catalog.json`.
2. Per **varianti e relativi collegamenti**, prevale `cdp-event-bindings.json`.
3. Questo documento definisce i **vincoli d'uso** di tali dati (per esempio il divieto di attivare automaticamente equivalenze).
4. Se emerge una contraddizione fra questi vincoli e il pacchetto, **non correggere silenziosamente i nomi canonici o le definizioni**: documentare la discrepanza e proporre una versione esplicita.

## 3. Struttura e provenienza del catalogo

| Insieme | Definizioni primarie |
|---|---:|
| General Website (`general_website`) | **17** |
| Ecommerce (`ecommerce`) | **36**: 28 ufficiali comuni alle specifiche ecommerce esaminate, 1 nome attestato in esempio, 7 proposte CDP |
| Moduli opzionali (`optional.*`) | **48** |
| Storico tecnico (`legacy.technical`) | **1** |
| **Totale** | **102** |

Classificazione della provenienza (campo `origin`):

| Classe | Definizioni | Significato |
|---|---:|---|
| `A. Official standard` | **71** | Definizione espressamente documentata in una specifica/API del provider; non implica che tutti i provider la prescrivano o che ogni sorgente la emetta |
| `B. Official recommendation` | **16** | Nome attestato in un esempio o raccomandazione ufficiale; non è necessariamente definito un contratto semantico completo |
| `C. CDP suggested` | **14** | Proposta del catalogo CDP secondo convenzioni di naming coerenti |
| `D. Historical / Legacy` | **1** | Compatibilità tecnica storica, disattivata salvo esplicita necessità |

I **16 binding** non costituiscono 16 nuove definizioni: sono varianti separate e inizialmente tutte disabilitate.

L'attributo `catalog` indica il **catalogo primario**. `related_catalogs` e `catalog_membership` consentono di esporre lo **stesso ID** in più sezioni senza duplicare il concetto (per esempio `Signed In` nei contesti generalista e B2B).

## 4. Contratto semantico di una definizione

Ogni definizione del seed contiene **20 campi**:

| Campo | Significato e vincolo |
|---|---|
| `id` | Identificatore semantico stabile; **non** dipende dalla traduzione né dal nome tecnico |
| `definition_version` | Versione della definizione; nel seed è `1` per tutte le 102 voci |
| `display_name` | Nome breve mostrato nella UI, in italiano |
| `description` | Significato dell'evento e limitazioni interpretative, in italiano |
| `category` | Categoria funzionale |
| `catalog` | Catalogo primario |
| `related_catalogs` | Altri cataloghi nei quali la stessa definizione è riutilizzabile |
| `synonyms` | Espressioni italiane equivalenti utili al riconoscimento semantico; **non** sono alias dei nomi tecnici |
| `match` | Espressione strutturata per confrontare l'envelope degli eventi |
| `origin` | Provenienza A/B/C/D, come definita sopra |
| `standard` | Attestazioni documentali: provider, specifica, versione, nome documentato, URL |
| `source_url` | Fonte principale; può essere `null` per una proposta CDP |
| `sources` | Fonti supplementari con ruolo dell'attestazione |
| `enabled_by_default` | Appartenenza al preset consigliato, **non** abilitazione della raccolta reale |
| `tier` | Importanza nel catalogo: fondamentale, opzionale, specialistico, storico |
| `semantic_kind` | Natura semantica: navigazione, azione utente, evento di sistema, aggiornamento identità |
| `technical_names` | Nomi canonici degli eventi `track`; normalmente vuoto per le primitive senza nome evento obbligatorio |
| `key_properties` | Proprietà significative con provenienza; **non** schema completo né dichiarazione di obbligatorietà |
| `recommended_business_key` | Chiave consigliata per distinguere entità di business, ove disponibile e verificata |
| `notes` | Ambiguità, vincoli interpretativi e osservazioni sulle integrazioni |

Una definizione non implica che il relativo evento sia emesso da una sorgente. Non trattare un nome tecnico documentato come prova di presenza o corretto tracciamento.

### Esempio canonico: acquisto completato

```json
{
  "id": "commerce.order_completed",
  "definition_version": 1,
  "display_name": "Acquisto completato",
  "match": {
    "any_of": [
      {"all": [
        {"field": "type", "op": "eq", "value": "track"},
        {"field": "event", "op": "eq", "value": "Order Completed"}
      ]}
    ]
  }
}
```

È un **estratto illustrativo**, non un sostituto dell'oggetto completo nel JSON. Non associare automaticamente `Order Created` o `Purchase Completed` a questa definizione.

## 5. Come riconoscere tecnicamente gli eventi

Il confronto si applica ai campi dell'**envelope normalizzato del CDP**, conservando i campi originali Segment/RudderStack. Per il seed 1.1.0 tutti i 102 matcher sono costituiti da un singolo ramo `any_of` con condizioni `all`, ma il formato supporta più rami.

| `type` del messaggio | Campo nome | Matching corretto | Cosa NON inferire |
|---|---|---|---|
| `track` | `event` richiesto | `type = track` **e** `event = nome canonico` | Non recuperare il nome da `name` o da `properties.name` |
| `page` | `name` opzionale | `type = page` per tutte le pagine; aggiungere `name` soltanto per una pagina specifica | Non inventare un `track` `Page Viewed` |
| `screen` | `name` opzionale nella specifica generale | `type = screen` per tutte le schermate | Non convertire automaticamente `screen` in `page` |
| `identify` | Nessun nome `event` | `type = identify` | Non equivale a login, registrazione o modifica effettiva del profilo |
| `group` | Nessun nome `event` | `type = group` | Non prova una prima iscrizione al gruppo o la creazione dell'account aziendale |
| `alias` | Nessun nome `event` | `type = alias` | Non è un alias del nome evento e non prova un merge universale dei profili |
| `merge` | Tipo storico RudderStack | Soltanto la definizione `legacy.identity_merge`, disabilitata nel preset | Non promuoverlo a tipo di evento corrente generalista |

**Regole obbligatorie di valutazione del matcher:**

- `any_of` è una disgiunzione logica (**OR**), `all` è una congiunzione (**AND**).
- `eq` significa **uguaglianza esatta e case-sensitive**. `Order Completed` e `order completed` non sono lo stesso valore.
- Se il campo testato non esiste, `eq` è **falso**.
- Nessun fallback automatico fra `event`, `name`, `properties.name`, `traits.name` o altre proprietà.
- Il nome visualizzato e i sinonimi italiani servono **solo a individuare una definizione**, non a verificare il messaggio ricevuto.
- Un singolo messaggio che soddisfa più rami OR della medesima definizione **appartiene una sola volta** a quell'evento semantico.
- Il matcher **classifica** messaggi; la verifica dell'integrità del payload, dell'identità, della provenienza e delle proprietà è un controllo distinto. Un matcher positivo, da solo, non certifica che un evento sia stato emesso nel momento corretto.

Esempio: `{"type":"page"}` corrisponde a `web.page_viewed`, anche senza `name`; `{"type":"track","event":"Page Viewed"}` **non** corrisponde a `web.page_viewed`, a meno che un amministratore configuri esplicitamente una diversa definizione o un proprio binding.

## 6. Convenzioni per i nuovi nomi tecnici

Per i nuovi eventi `track` non già documentati, applicare la convenzione Segment **Object + Action**, nome in **Title Case**, preferibilmente azione al passato e nome stabile, per esempio `Order Completed`, `Content Shared`, `Newsletter Subscribed`.

- Conservare **esattamente** i nomi ufficiali, anche quando non rispettano una regola uniforme: `Signed In`, `Products Searched`, `Account Added User`, `Product Added to Wishlist`, `Unsubscribed`, `Video Content Playing`.
- Non inserire valori dinamici nei nomi evento: usare proprietà per ID prodotto, piano, categoria, coupon o query.
- Per le **nuove** proprietà preferire `snake_case`; non riscrivere campi ufficiali con altre convenzioni, per esempio `userId`, `context.groupId`, `startupTime`, `droppedFrames`.
- Non dedurre dalla correttezza grammaticale del nome che si tratti di uno standard Segment/RudderStack.
- Per una nuova definizione locale, salvare esplicitamente la provenienza **custom del workspace**, distinta dalle classi A/B/C/D dello **specifico seed**. Se lo schema del seed non ammette questo valore, mantenerlo in metadati locali separati, non dichiarare l'evento come standard ufficiale o raccomandazione.

## 7. Cataloghi disponibili e selezione predefinita

Nel catalogo di riferimento esistono tutte le 102 definizioni; non tutte devono essere presentate come attive in ogni installazione.

Il preset generalista contiene **8 definizioni**, condizionate alle funzionalità presenti: `web.page_viewed`, `identity.identified`, `auth.signed_up`, `auth.signed_in`, `auth.signed_out`, `web.search_submitted`, `form.submitted`, `lead.captured`.

L'addon ecommerce contiene **4 definizioni**: `commerce.product_viewed`, `commerce.product_added`, `commerce.checkout_started`, `commerce.order_completed`.

Gli altri eventi sono disponibili nel catalogo di riferimento e possono essere attivati quando pertinenti. I moduli opzionali (`optional.*`) e lo storico (`legacy.*`) sono inizialmente disattivati. Non abilitare un evento di login se il sito non supporta un login; non abilitare automaticamente eventi di wishlist, email o video per un sito che non li raccoglie.

Tenere concettualmente distinti:

| Stato | Significato |
|---|---|
| Disponibile | La definizione esiste nel catalogo |
| Abilitato | Può essere selezionata per nuove interrogazioni |
| Configurato | Esiste una corrispondenza con la sorgente del workspace |
| Osservato | Sono stati effettivamente ricevuti messaggi corrispondenti |
| Verificato | Il contratto di emissione è stato controllato |
| Copertura nota | È noto fino a quale periodo/sorgente i dati sono affidabili |

`enabled_by_default: true` indica **solo una proposta di preset**. Non prova che un evento sia configurato, osservato o misurabile. In particolare, un evento non configurato **non equivale** a «zero azioni compiute».

Per la selezione nella UI, privilegiare `display_name`, `description`, `category`, `catalog`, `synonyms` e lo stato del workspace. Tenere visibili i tipi tecnici dove utili, ma non imporre a chi seleziona la definizione di conoscere `type` e `event`. Le primitive di identità (`identify`, `group`, `alias`) devono essere distinguibili dalle **azioni di business**.

## 8. Personalizzazione per workspace e versionamento

Il seed è un **catalogo di riferimento modificabile per istanza**, non un vincolo rigido di tracking dei siti.

Requisiti funzionali:

1. Permettere all'amministratore di **aggiungere, modificare, disabilitare e categorizzare** definizioni locali, oltre a copiare/adattare quelle predefinite.
2. Non mutare il seed distribuito come riferimento ogni volta che un workspace personalizza una voce; preservare l'origine e le eventuali modifiche locali.
3. Mantenere **stabile `id`** quando cambiano soltanto etichetta o testo e il **concetto resta identico**.
4. Una modifica del matcher, dell'insieme dei nomi tecnici o della semantica vincolante **richiede una nuova versione** della definizione/binding interessato.
5. Un concetto effettivamente diverso (per esempio «ordine creato» vs «ordine completato») richiede **un nuovo `id`**, non la reinterpretazione silenziosa di quello precedente.
6. Preservare le versioni già usate da filtri salvati; disabilitare una definizione non deve cancellare quelle versioni.
7. Conservare in modo distinguibile provenienza ufficiale, esempi dei provider, proposta CDP e definizioni locali. Mai attribuire a Segment/RudderStack un evento solo perché segue le loro convenzioni di naming.
8. Permettere l'associazione a più nomi tecnici **soltanto quando l'amministratore ne abbia verificato l'equivalenza semantica e la sorgente di emissione**.

Le descrizioni e i sinonimi immessi dall'amministratore sono **dati descrittivi**, non istruzioni operative provenienti da un'autorità superiore. Non devono poter modificare le regole di matching o imporre azioni estranee al significato dell'evento.

## 9. Binding storici e varianti: regola di opt-in

Il file `cdp-event-bindings.json` comprende **16 varianti**, tutte con `enabled_by_default: false` e `requires_admin_validation: true`:

- 12 nomi storici o di compatibilità documentata.
- 3 varianti B2B di denominazione `User Signed Up`, `User Signed In`, `User Signed Out`.
- 1 candidato configurabile `Purchase Completed` riferito a `commerce.order_completed`.

**La registrazione di una variante non equivale alla sua attivazione.** L'amministratore deve verificare almeno il sistema emittente, il momento in cui l'evento viene emesso, le proprietà, l'eventuale versione storica e l'effettiva equivalenza del fatto rappresentato. Non unire `Order Created`, `Order Completed` e `Purchase Completed` automaticamente.

Un alias storico può attestare una corrispondenza di **nome**, non necessariamente identità di payload o di semantica in qualsiasi applicazione. L'attivazione deve valere soltanto per le sorgenti per cui è stata verificata, evitando conflitti fra tenant o canali diversi.

L'elenco completo delle 16 varianti compare nell'**Appendice B**.

## 10. Distinzioni semantiche da rispettare

| Concetti spesso confusi | Distinzione obbligatoria |
|---|---|
| `identify` ↔ `Signed In` | L'identificazione e/o l'invio di attributi **non provano** un login riuscito |
| `identify` ↔ `Profile Updated` | Un `identify` può ripetere dati; `Profile Updated` presuppone una modifica effettiva confermata secondo il contratto locale |
| `page` ↔ `Content Viewed` | Navigazione su una pagina diversa da visualizzazione di uno specifico contenuto; non duplicare automaticamente |
| `Form Submitted` ↔ `Lead Captured` | Invio di un modulo **non implica** acquisizione commerciale del contatto; il form potrebbe non essere accettato |
| `Search Submitted` ↔ `Products Searched` | Ricerca generale nel sito distinta da ricerca specifica nel catalogo ecommerce |
| `Checkout Started` ↔ `Order Created` | Avvio del checkout distinto dalla persistenza di un'entità ordine |
| `Order Created` ↔ `Order Completed` | Creazione di un ordine **non implica** acquisto completato |
| `Payment Info Entered` ↔ `Payment Authorized` ↔ `Payment Captured` | Inserimento dei dati, autorizzazione e cattura sono passaggi differenti; non equivalgono universalmente a un pagamento definitivamente ricevuto |
| `Order Refunded` ↔ `Refund Requested` | Rimborso eseguito distinto da mera richiesta |
| `Order Cancelled` ↔ `Subscription Cancelled` | Annullamento dell'ordine diverso da cessazione dell'abbonamento |
| `Unsubscribed` ↔ `Subscription Cancelled` | Disiscrizione dalle comunicazioni marketing diversa dalla cancellazione di un abbonamento commerciale |
| `Email Delivered` ↔ `Email Opened` | Consegna a un server non significa lettura umana; anche il tracciamento di apertura ha limiti propri |
| `Video Content Playing` ↔ `Video Content Started` | `Playing` può essere un segnale periodico; non conteggiare ogni heartbeat come una nuova visualizzazione |
| `Product Added` ↔ carrello abbandonato | `Cart Abandoned` non è fra i 28 eventi ecommerce canonici: l'abbandono è generalmente una **condizione derivata** da eventi e tempo |
| `alias` ↔ `merge` | `alias` riguarda gli identificatori; non garantisce da solo unione di profili. `merge` è storico e non va usato come default |

La descrizione dell'evento deve esprimere **che cosa attesta** e **che cosa non attesta**. Una descrizione più attraente, ma semanticamente inesatta, è un errore di catalogo.

## 11. Proprietà degli eventi e differenze Segment/RudderStack

- La corrispondenza `track` usa **`event` alla radice**; `properties.name` può essere il nome di un prodotto o di un'altra entità, non il nome dell'evento.
- `page` e `screen` possono avere `name` opzionale. Per una voce generica `web.page_viewed` o `app.screen_viewed` è sufficiente `type`.
- Per `identify`, Segment documenta `traits` alla radice; RudderStack utilizza normalmente `context.traits`, con eccezioni di compatibilità per alcuni SDK. Conservare l'origine del dato; non inventare fallback automatici per il nome dell'evento.
- Nei messaggi `group`, i `traits` del gruppo non devono diventare automaticamente attributi della persona; `context.groupId` nei messaggi `track` non è un messaggio di tipo `group`.
- Le proprietà ecommerce dei due provider **non sono identiche**: `cart_id`, `checkout_id`, campi monetari, lista `products[]`, `rating` e coupon richiedono attenzione al contratto della sorgente.
- Per le proprietà annidate in `products[]`, gli attributi che descrivono lo stesso prodotto devono essere letti **sullo stesso elemento**; non associare l'ID di un prodotto alla categoria di un altro.
- `key_properties` indica proprietà **utili**, non necessariamente **obbligatorie**. `recommended_business_key` non prova che la proprietà sia presente o univoca nei dati di produzione.
- Distinguere il **messaggio** da una **entità di business**: due eventi `Order Completed` con diversi message ID possono riferirsi allo stesso `order_id`; tre `Product Viewed` sullo stesso `product_id` sono tre visualizzazioni ma un solo prodotto distinto.
- Per conoscere se un'azione è avvenuta durante un periodo, è rilevante il suo **istante di evento**, non necessariamente l'istante di ricezione. L'assenza di messaggi nel periodo non dimostra da sola l'assenza dell'azione senza copertura affidabile e corretta associazione al profilo.

Questi vincoli delimitano il **significato del dato evento**; non prescrivono un motore di query o una DSL.

## 12. Criteri di accettazione sul catalogo eventi

Considerare completata la parte relativa agli eventi quando, almeno:

1. Le **102** definizioni sono caricate senza perdita di informazioni, con `id` e `definition_version` originali e **nessun duplicato**.
2. Il numero di definizioni per catalogo e origine coincide con il JSON, senza creare copie per i cataloghi correlati.
3. I **28** nomi ecommerce canonici sono conservati esattamente come nel seed e sono di tipo `track`.
4. Tutte le **16** varianti sono disponibili nel registro separato ma **disabilitate** per impostazione predefinita.
5. Il matching `track` verifica `type` e `event` e non si attiva soltanto sulla base di `name` o `properties.name`.
6. Il matching `page` e `screen` riconosce il tipo anche con `name` assente.
7. Il matching `identify`, `group`, `alias` non richiede né inventa un nome `event`.
8. Il confronto dei valori tecnici è esatto e sensibile alle maiuscole; i campi mancanti non soddisfano `eq`.
9. Una singola istanza non viene contata più volte perché soddisfa più alternative dello stesso matcher.
10. `Order Created`, `Order Completed`, `Purchase Completed`, `Signed In` e `identify` **non** vengono unificati automaticamente.
11. Il sistema conserva etichetta, descrizione, sinonimi e provenienza senza confondere i sinonimi italiani con alias tecnici.
12. Le definizioni locali sono modificabili senza alterare retroattivamente versioni già referenziate e senza spacciare eventi custom per standard ufficiali.
13. Disponibilità del catalogo, abilitazione nel workspace, configurazione, osservazione e verifica rimangono concetti distinti.
14. Una definizione non configurata **non** è trattata come assenza di evento.
15. Il catalogo e il registro binding rispettano i relativi schemi JSON; i casi di `fixtures/matcher-cases.json` rimangono validi.

La suite completa del pacchetto dichiara **202 controlli superati**; include anche test non relativi ai soli eventi. Non scambiare tale risultato con una verifica di dati reali del CDP.

---

## Appendice A — Inventario completo delle 102 definizioni

Di seguito sono riportati, **senza traduzioni o rinominazioni dei campi tecnici**, i valori effettivi di `id`, matcher canonico, `display_name`, `description`, `synonyms`, `origin` e `enabled_by_default` estratti dal JSON. Il documento resta consultabile autonomamente per semantica e identificazione; tutti gli altri metadati, compresi i riferimenti di provenienza per singola voce e le proprietà, rimangono nel JSON autorevole.

**Legenda:** A = standard documentato; B = esempio/raccomandazione ufficiale; C = proposta CDP; D = storico. `Preset = Sì` indica il valore del seed, **non** disponibilità dei dati nel workspace.

### A. General Website — 17 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `auth.signed_in` | `track` / `Signed In` | Accesso effettuato | A | Sì | L’utente ha effettuato l’accesso al servizio. Non rappresenta un tentativo fallito, una pagina di login visitata o la riassociazione di un utente già autenticato tramite identify. | login; ha effettuato l’accesso; si è autenticato |
| `auth.signed_out` | `track` / `Signed Out` | Uscita effettuata | A | Sì | L’utente ha eseguito l’uscita dal servizio. Non inferisce scadenza della sessione, chiusura del browser o inattività, salvo un distinto contratto locale che documenti tali casi. | logout; ha effettuato l’uscita; si è disconnesso |
| `auth.signed_up` | `track` / `Signed Up` | Registrazione completata | A | Sì | L’utente si è registrato al servizio. Misura la registrazione, non la visita alla pagina di iscrizione o il semplice invio di identify. Non garantisce verifica email o attivazione di un piano a pagamento. | registrazione; nuovo utente; si è iscritto al sito |
| `form.submitted` | `track` / `Form Submitted` | Modulo inviato | B | Sì | È stato rilevato l’invio di un modulo. La raccomandazione ufficiale non stabilisce se il server lo abbia accettato: per richiedere successo usare una proprietà validata o un evento business specifico. Non equivale automaticamente a lead acquisito. | modulo inviato; form inviato; ha compilato e inviato |
| `identity.identified` | `identify` (qualsiasi `name`) | Identità e tratti registrati | A | Sì | È stata ricevuta una chiamata identify che associa identificatori e può trasmettere attributi di una persona. Può ripetere dati già noti o avvenire al caricamento pagina. Non dimostra un login, una nuova registrazione o una modifica effettiva del profilo. | utente identificato; tratti ricevuti; identificazione del profilo |
| `lead.captured` | `track` / `Lead Captured` | Contatto commerciale acquisito | B | Sì | È stata registrata l’acquisizione dei dati di un potenziale cliente. Non garantisce qualificazione commerciale, appuntamento, acquisto o consenso marketing; questi fatti richiedono segnali e attributi separati. | lead acquisito; nuovo contatto commerciale; richiesta di contatto acquisita |
| `web.page_viewed` | `page` (qualsiasi `name`) | Pagina visualizzata | A | Sì | È stata registrata la visualizzazione di una pagina web, anche quando la chiamata non contiene un nome. Consente filtri di navigazione per URL o pagina; non certifica una sessione distinta, la lettura del contenuto o una conversione. | pagina vista; visita a una pagina; navigazione sul sito |
| `web.search_submitted` | `track` / `Search Submitted` | Ricerca interna inviata | C | Sì | L’utente ha inviato una query alla ricerca interna del sito. Non garantisce risultati disponibili, clic sui risultati o ricerca di prodotti. Per il catalogo prodotti usare commerce.products_searched; l’ambito deve evitare duplicazioni. | ricerca sul sito; ha cercato nel sito; query inviata |
| `communication.email_unsubscribed` | `track` / `Unsubscribed` | Disiscrizione dalle email | A | No | È stata registrata la rinuncia a ricevere email marketing nell’ambito dichiarato dal provider: una lista oppure tutte le email marketing. Non è la cancellazione di un abbonamento a pagamento, non riguarda automaticamente email transazionali e non dimostra revoca di tutti i consensi su ogni canale. | disiscritto dalle email; opt-out email; disiscrizione newsletter |
| `communication.newsletter_subscribed` | `track` / `Newsletter Subscribed` | Iscrizione newsletter confermata | C | No | Il sistema di iscrizione conferma l’inserimento nella newsletter o lista indicata. Per un flusso con conferma via email, emettere dopo tale conferma. Un tentativo di invio modulo non basta; non è un abbonamento commerciale. | iscritto alla newsletter; iscrizione email confermata |
| `content.shared` | `track` / `Content Shared` | Contenuto condiviso | B | No | È stata registrata una condivisione di contenuto secondo il punto di rilevazione locale. Il nome da solo non garantisce pubblicazione sulla piattaforma esterna, consegna al destinatario o lettura. | ha condiviso un contenuto; condivisione articolo |
| `content.viewed` | `track` / `Content Viewed` | Contenuto visualizzato | B | No | È stata registrata la visualizzazione di un contenuto identificabile, per esempio un articolo. Non equivale a lettura completa. Usare questo evento solo se la visualizzazione del contenuto aggiunge informazione rispetto alla page view. | ha visto un contenuto; articolo visualizzato |
| `feature.used` | `track` / `Feature Used` | Funzionalità utilizzata | B | No | È stato registrato l’uso di una funzionalità identificata nel tracking plan. Occorre dichiarare quale operazione costituisce utilizzo: il nome ufficiale di esempio non garantisce successo dell’operazione né creazione di un risultato. | ha usato una funzione; utilizzo funzionalità |
| `form.initiated` | `track` / `Form Initiated` | Compilazione modulo iniziata | B | No | È iniziata l’interazione con un modulo secondo la soglia di avvio dichiarata nel tracking plan. Non basta la presenza del modulo nella pagina; non prova invio o acquisizione di un contatto. | ha iniziato un modulo; compilazione iniziata |
| `interaction.button_clicked` | `track` / `Button Clicked` | Pulsante selezionato | B | No | L’utente ha attivato un pulsante selezionato dal tracking plan. Non indica automaticamente invio riuscito, acquisto, autenticazione o completamento della funzione associata. | clic su pulsante; bottone premuto |
| `interaction.link_clicked` | `track` / `Link Clicked` | Link selezionato | B | No | L’utente ha attivato un collegamento. Questo misura l’interazione col link, senza provare che la destinazione sia stata caricata o che l’azione di business collegata sia riuscita. | clic su link; collegamento selezionato |
| `profile.updated` | `track` / `Profile Updated` | Profilo modificato | C | No | Il sistema conferma il salvataggio di una modifica effettiva ai dati del profilo. Non usare questo evento per una semplice ripetizione di identify o per la sola apertura del modulo del profilo. | profilo aggiornato; dati personali modificati |

### B. Ecommerce — 36 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `commerce.product_added` | `track` / `Product Added` | Prodotto aggiunto al carrello | A | Sì | Un prodotto è stato aggiunto al carrello. Non è la creazione di un prodotto nel catalogo, l’aggiunta alla wishlist o il completamento di un acquisto. quantity descrive la quantità secondo il contratto locale. | aggiunta al carrello; ha messo prodotti nel carrello |
| `commerce.checkout_started` | `track` / `Checkout Started` | Checkout iniziato | A | Sì | È stato avviato il percorso di checkout per un ordine o transazione. La spec Segment colloca il segnale all’inizio del checkout e parla di transazione creata; non garantisce un ordine confermato o un pagamento. | inizio checkout; ha iniziato l’acquisto; procedura di acquisto avviata |
| `commerce.order_completed` | `track` / `Order Completed` | Ordine completato | A | Sì | È stata registrata la conclusione riuscita di un ordine o transazione. È il segnale ecommerce canonico per un acquisto concluso; non garantisce settlement bancario, consegna o assenza di successivi annullamenti e rimborsi. | acquisto completato; ha acquistato; ordine concluso; ha effettuato acquisti |
| `commerce.product_viewed` | `track` / `Product Viewed` | Dettaglio prodotto visualizzato | A | Sì | È stata registrata la visualizzazione dei dettagli di un prodotto, anche in una finestra o schermata. Non è una semplice impressione in una lista né un acquisto. Ripetizioni sullo stesso product_id contano come più eventi, non più prodotti distinti. | ha visualizzato prodotti; scheda prodotto vista; dettaglio prodotto aperto |
| `commerce.product_list_filtered` | `track` / `Product List Filtered` | Lista prodotti filtrata | A | No | È stata applicata una selezione di filtri alla lista o categoria prodotti. filters e sorts descrivono filtri e ordinamenti; non equivale a ricerca testuale o dettaglio prodotto. | filtri prodotti applicati; lista filtrata |
| `commerce.product_list_viewed` | `track` / `Product List Viewed` | Lista prodotti visualizzata | A | No | È stata visualizzata una lista o categoria di prodotti. La presenza di prodotti nell’array non prova che siano stati aperti i relativi dettagli. | categoria visitata; lista prodotti vista |
| `commerce.products_searched` | `track` / `Products Searched` | Prodotti cercati | A | No | Ricerca nel catalogo prodotti; non generica ricerca editoriale né clic su un risultato. Il testo o oggetto query precisa ciò che è stato cercato. | ha cercato prodotti; ricerca prodotti |
| `commerce.cart_viewed` | `track` / `Cart Viewed` | Carrello visualizzato | A | No | È stato visualizzato il carrello e può esserne riportato il contenuto. Non prova l’avvio del checkout o un acquisto. | ha visto il carrello; carrello aperto |
| `commerce.product_removed` | `track` / `Product Removed` | Prodotto rimosso dal carrello | A | No | Un prodotto è stato rimosso dal carrello. Non indica eliminazione dal catalogo, annullamento di ordine o reso di merce già acquistata. | rimozione dal carrello; ha tolto un prodotto dal carrello |
| `commerce.checkout_step_completed` | `track` / `Checkout Step Completed` | Passaggio checkout completato | A | No | È stato completato uno specifico passaggio del checkout. Anche l’ultimo passaggio non sostituisce l’evento Order Completed. | fase checkout completata; passaggio checkout concluso |
| `commerce.checkout_step_viewed` | `track` / `Checkout Step Viewed` | Passaggio checkout visualizzato | A | No | È stato visualizzato uno specifico passaggio del checkout, identificato da step e checkout_id. Non dimostra completamento del passaggio o dell’ordine. | passaggio checkout visto; fase checkout visualizzata |
| `commerce.coupon_applied` | `track` / `Coupon Applied` | Coupon applicato | A | No | Un coupon è stato applicato al carrello o all’ordine. Non prova che l’ordine sia stato completato o che la promozione abbia generato un acquisto. | sconto applicato; coupon accettato |
| `commerce.coupon_denied` | `track` / `Coupon Denied` | Coupon rifiutato | A | No | Il sistema ha rifiutato un coupon per il carrello o l’ordine. reason può distinguere i motivi; non è una rimozione volontaria di un coupon valido. | coupon non valido; codice sconto rifiutato |
| `commerce.coupon_entered` | `track` / `Coupon Entered` | Coupon inserito | A | No | Un codice coupon è stato inserito nel carrello o nell’ordine. Non significa che il codice sia valido o che lo sconto sia stato applicato. | codice sconto inserito; coupon digitato |
| `commerce.coupon_removed` | `track` / `Coupon Removed` | Coupon rimosso | A | No | Un coupon è stato rimosso dal carrello o dall’ordine. Non indica automaticamente invalidità del codice o rimborso dello sconto. | coupon eliminato; codice sconto rimosso |
| `commerce.product_shipped` | `track` / `Product Shipped` | Prodotto spedito | B | No | È stata registrata la spedizione di un prodotto secondo il contratto logistico locale. Non garantisce consegna né spedizione di tutte le righe dell’ordine; l’esempio SDK non definisce una tassonomia completa di evasione. | prodotto spedito; articolo affidato al corriere |
| `commerce.order_cancelled` | `track` / `Order Cancelled` | Ordine annullato | A | No | È stato registrato l’annullamento di un ordine. Non prova che sia stato effettuato un rimborso o che la merce sia stata restituita. | ordine cancellato; annullamento ordine |
| `commerce.order_created` | `track` / `Order Created` | Ordine creato | C | No | Il sistema ha creato una nuova entità ordine persistita. L’ordine può essere in bozza o in attesa di conferma o pagamento. Non equivale a checkout iniziato, ordine completato o incasso. | ordine creato; creazione ordine |
| `commerce.order_updated` | `track` / `Order Updated` | Ordine modificato | A | No | È stato aggiornato un ordine o una transazione esistente. Non è un nuovo acquisto; il numero delle modifiche non misura il numero degli ordini. | ordine modificato; ordine aggiornato |
| `commerce.payment_authorized` | `track` / `Payment Authorized` | Pagamento autorizzato | C | No | Il sistema di pagamento conferma l’autorizzazione per uno specifico tentativo e importo. L’autorizzazione non è la cattura dei fondi e può scadere o essere revocata. | pagamento autorizzato; autorizzazione carta |
| `commerce.payment_captured` | `track` / `Payment Captured` | Pagamento acquisito | C | No | Il sistema di pagamento conferma la cattura di uno specifico importo. Non rappresenta il solo inserimento della carta o l’autorizzazione; la disponibilità finale dei fondi segue il contratto del provider. | pagamento catturato; addebito acquisito |
| `commerce.payment_failed` | `track` / `Payment Failed` | Tentativo di pagamento fallito | C | No | Un tentativo di pagamento ha un esito negativo confermato dal sistema. L’ordine potrebbe essere pagato con un tentativo successivo; l’assenza di Payment Captured non basta a generare questo evento. | pagamento fallito; addebito rifiutato |
| `commerce.payment_info_entered` | `track` / `Payment Info Entered` | Dati di pagamento inseriti | A | No | Sono stati inseriti con successo i dati necessari al pagamento. Non attesta autorizzazione, addebito, acquisizione dei fondi o completamento dell’ordine. | dati pagamento inseriti; metodo di pagamento inserito |
| `commerce.product_clicked` | `track` / `Product Clicked` | Prodotto selezionato | A | No | L’utente ha selezionato un prodotto con un clic. Non è una garanzia di caricamento dei dettagli, che appartengono a Product Viewed. | clic su prodotto; prodotto selezionato |
| `commerce.promotion_clicked` | `track` / `Promotion Clicked` | Promozione selezionata | A | No | È stato rilevato un clic sulla promozione. Non garantisce visualizzazione della destinazione, sconto applicato o conversione. | clic sulla promozione; banner promozionale selezionato |
| `commerce.promotion_viewed` | `track` / `Promotion Viewed` | Promozione visualizzata | A | No | È stata rilevata una visualizzazione della promozione. Non prova clic, applicazione di coupon o acquisto. | promozione vista; banner promozionale visualizzato |
| `commerce.order_refunded` | `track` / `Order Refunded` | Ordine rimborsato | A | No | È stato registrato un rimborso relativo a un ordine. Non è la sola richiesta di rimborso né un reso fisico; il solo evento non distingue in modo normativo rimborso totale e parziale. | rimborso effettuato; ordine rimborsato |
| `commerce.refund_requested` | `track` / `Refund Requested` | Rimborso richiesto | C | No | È stata registrata una richiesta di rimborso economico. Non attesta che il rimborso sia stato approvato, eseguito o accreditato. Non richiede necessariamente un reso fisico. | richiesta rimborso; ha chiesto un rimborso |
| `commerce.return_received` | `track` / `Return Received` | Reso ricevuto | C | No | Il sistema logistico conferma la ricezione fisica della merce restituita. Non prova approvazione finale o rimborso. | merce restituita ricevuta; reso arrivato |
| `commerce.return_requested` | `track` / `Return Requested` | Reso richiesto | C | No | È stata presentata una richiesta di restituzione della merce relativa a un ordine. Non attesta autorizzazione del reso, ricezione del prodotto o rimborso. | richiesta di reso; vuole restituire prodotti |
| `commerce.product_reviewed` | `track` / `Product Reviewed` | Prodotto recensito | A | No | È stata registrata una recensione di prodotto. Il nome non garantisce pubblicazione dopo moderazione né una specifica scala numerica per rating. | ha recensito un prodotto; recensione prodotto inviata |
| `commerce.cart_shared` | `track` / `Cart Shared` | Carrello condiviso | A | No | È stata registrata la condivisione del carrello. Non prova che un destinatario lo abbia aperto, copiato o acquistato. | ha condiviso il carrello; carrello inviato |
| `commerce.product_shared` | `track` / `Product Shared` | Prodotto condiviso | A | No | È stata registrata la condivisione di un prodotto con uno o più destinatari. Non prova ricezione, apertura o acquisto da parte dei destinatari. | ha condiviso un prodotto; prodotto inviato a un amico |
| `commerce.product_added_to_wishlist` | `track` / `Product Added to Wishlist` | Prodotto aggiunto ai desideri | A | No | Un prodotto è stato aggiunto alla lista desideri. Non è stato necessariamente aggiunto al carrello o acquistato. | aggiunto alla wishlist; prodotto desiderato |
| `commerce.product_removed_from_wishlist` | `track` / `Product Removed from Wishlist` | Prodotto rimosso dai desideri | A | No | Un prodotto è stato rimosso dalla lista desideri. Non equivale a rimozione dal carrello o abbandono di un acquisto. | rimosso dalla wishlist; eliminato dai desideri |
| `commerce.wishlist_product_added_to_cart` | `track` / `Wishlist Product Added to Cart` | Prodotto dai desideri al carrello | A | No | Un prodotto presente nella wishlist è stato aggiunto al carrello. Non garantisce la sua rimozione dalla wishlist. Può sovrapporsi a Product Added nella stessa implementazione: evitare doppio conteggio della medesima aggiunta. | wishlist nel carrello; aggiunta dalla lista desideri |

### C1. B2B SaaS — 8 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `b2b.account_added_user` | `track` / `Account Added User` | Utente aggiunto all’account | A | No | Una persona è stata aggiunta a un account aziendale. L’evento riguarda la relazione di appartenenza; non implica una nuova registrazione globale o identifica necessariamente l’amministratore che ha effettuato l’aggiunta. | ingresso nel team; utente aggiunto all’organizzazione |
| `b2b.account_created` | `track` / `Account Created` | Account aziendale creato | A | No | È stato creato un account aziendale o organizzazione. Non è necessariamente la registrazione di una persona. | organizzazione creata; nuovo account aziendale |
| `b2b.account_deleted` | `track` / `Account Deleted` | Account aziendale eliminato | A | No | È stato eliminato un account aziendale. Non implica eliminazione di tutti i profili delle persone associate o cancellazione dei relativi eventi. | organizzazione eliminata; account aziendale cancellato |
| `b2b.account_removed_user` | `track` / `Account Removed User` | Utente rimosso dall’account | A | No | Una persona è stata rimossa da un account aziendale. Non implica eliminazione della sua identità o perdita di accesso ad altri account. | uscita dal team; utente rimosso dall’organizzazione |
| `b2b.invite_sent` | `track` / `Invite Sent` | Invito inviato | A | No | È stato inviato un invito a entrare nell’account. Non prova ricezione, accettazione o ingresso dell’invitato. Il profilo attore e l’invitato possono essere persone diverse. | invito a un collega; utente invitato |
| `identity.group_associated` | `group` (qualsiasi `name`) | Associazione a un gruppo registrata | A | No | È stata ricevuta una chiamata group che collega una persona a un gruppo e può trasmettere attributi del gruppo. Non attesta la prima aggiunta della persona, la creazione dell’account o una modifica effettiva dell’appartenenza. | associazione al gruppo; tratti aziendali registrati |
| `b2b.trial_ended` | `track` / `Trial Ended` | Prova terminata | A | No | È terminato il periodo di prova. Il solo evento non distingue conversione a pagamento, scadenza senza acquisto o altra causa. | trial finito; periodo di prova terminato |
| `b2b.trial_started` | `track` / `Trial Started` | Prova iniziata | A | No | È iniziato un periodo di prova del servizio o piano. Non indica sottoscrizione pagante; gruppo, piano e date delimitano il contesto della prova. | periodo di prova iniziato; trial avviato |

### C2. Content — 5 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `content.article_bookmarked` | `track` / `Article Bookmarked` | Articolo salvato nei preferiti | B | No | È stata registrata l’aggiunta di un articolo ai preferiti. Non implica lettura completa o condivisione. | articolo salvato; segnalibro aggiunto |
| `content.article_completed` | `track` / `Article Completed` | Articolo completato | B | No | È stato raggiunto il criterio locale di completamento di un articolo. La fonte ufficiale non definisce una soglia di tempo o scorrimento e non prova comprensione del testo. | articolo letto fino alla fine; lettura completata |
| `content.comment_submitted` | `track` / `Comment Submitted` | Commento inviato | B | No | È stato inviato un commento su un contenuto. Non garantisce pubblicazione, approvazione della moderazione o lettura da altri utenti. | commento inviato; ha commentato |
| `content.content_curated` | `track` / `Content Curated` | Contenuto selezionato editorialmente | B | No | È stata registrata un’azione di selezione o cura editoriale di un contenuto. Non equivale a semplice visualizzazione, produzione originale o condivisione. | contenuto curato; selezione editoriale |
| `content.content_produced` | `track` / `Content Produced` | Contenuto prodotto | B | No | È stata registrata la produzione di un contenuto. Il tracking plan deve distinguere bozza, salvataggio e pubblicazione; il nome di esempio non garantisce visibilità pubblica. | contenuto creato; autore di un contenuto |

### C3. Video — 17 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `video.ad_completed` | `track` / `Video Ad Completed` | Pubblicità video completata | A | No | Termina la riproduzione dell’asset pubblicitario. Non equivale a completamento del contenuto principale o clic sulla pubblicità. | spot completato; pubblicità video terminata |
| `video.ad_playing` | `track` / `Video Ad Playing` | Avanzamento pubblicità video | A | No | Avanzamento periodico della riproduzione pubblicitaria. È un aggiornamento periodico: non contare ogni messaggio come nuova impressione pubblicitaria. | spot in riproduzione; heartbeat pubblicità |
| `video.ad_started` | `track` / `Video Ad Started` | Pubblicità video iniziata | A | No | Inizia un asset pubblicitario nella riproduzione. Non prova che lo spot sia stato completato o cliccato. | spot iniziato; inizio pubblicità video |
| `video.content_completed` | `track` / `Video Content Completed` | Contenuto video completato | A | No | Termina un segmento di contenuto; posizione e durata coincidono. Non prova che tutto il video sia stato guardato senza salti; un asset può essere ripartito in segmenti. | segmento video completato; contenuto video terminato |
| `video.content_playing` | `track` / `Video Content Playing` | Avanzamento contenuto video | A | No | Heartbeat periodico con avanzamento della riproduzione del contenuto. È un segnale periodico, non una nuova visualizzazione: il conteggio grezzo misura heartbeat. | heartbeat video; riproduzione in corso |
| `video.content_started` | `track` / `Video Content Started` | Contenuto video iniziato | A | No | Inizia un segmento di contenuto nella sessione. È distinto dall’avvio del player e dall’inizio della pubblicità. | contenuto video avviato; inizio segmento video |
| `video.playback_buffer_completed` | `track` / `Video Playback Buffer Completed` | Buffering video terminato | A | No | Termina il buffering del player. Non è il completamento della riproduzione. | buffering terminato; buffer pronto |
| `video.playback_buffer_started` | `track` / `Video Playback Buffer Started` | Buffering video iniziato | A | No | Il player inizia il buffering di contenuto o pubblicità. Non è un nuovo avvio di contenuto o una nuova visualizzazione. | buffering iniziato; attesa dati video |
| `video.playback_completed` | `track` / `Video Playback Completed` | Sessione video completata | A | No | Termina l’intera sessione di riproduzione del player. È la fine della sessione del player, distinta dalla fine del singolo contenuto. | sessione player completata; fine riproduzione |
| `video.playback_exited` | `track` / `Video Playback Exited` | Riproduzione video abbandonata | A | No | L’utente abbandona la riproduzione o lo stream. Non implica completamento del contenuto; evento documentato qui soltanto da Segment. | uscita dal video; abbandono player |
| `video.playback_interrupted` | `track` / `Video Playback Interrupted` | Riproduzione video interrotta | A | No | La riproduzione si interrompe involontariamente. Non attribuire l’interruzione a una scelta volontaria di pausa. | interruzione video; riproduzione interrotta |
| `video.playback_paused` | `track` / `Video Playback Paused` | Riproduzione video in pausa | A | No | L’utente mette in pausa il player. È una pausa; non equivale a un’interruzione involontaria o all’uscita dal player. | video in pausa; pausa del player |
| `video.playback_resumed` | `track` / `Video Playback Resumed` | Riproduzione video ripresa | A | No | L’utente riprende dopo una pausa. Non contare la ripresa come un nuovo inizio di sessione. | video ripreso; ripresa dopo pausa |
| `video.playback_seek_completed` | `track` / `Video Playback Seek Completed` | Spostamento nel video completato | A | No | Termina lo spostamento manuale nella riproduzione. Non dimostra che il contenuto saltato sia stato guardato. | posizione video raggiunta; spostamento completato |
| `video.playback_seek_started` | `track` / `Video Playback Seek Started` | Spostamento nel video iniziato | A | No | L’utente avvia spostamento manuale nella riproduzione. La posizione richiesta non è ancora necessariamente raggiunta. | ricerca posizione video; spostamento iniziato |
| `video.playback_started` | `track` / `Video Playback Started` | Sessione video iniziata | A | No | Avvio della sessione del player dopo l’ultima azione utente necessaria alla riproduzione. Non garantisce che sia iniziato il contenuto principale: la sessione può iniziare con pubblicità. | avvio player; sessione video avviata |
| `video.quality_updated` | `track` / `Video Quality Updated` | Qualità video aggiornata | A | No | Sono cambiati parametri di qualità della riproduzione video. È telemetria tecnica del player, non una visualizzazione o un’azione commerciale. | qualità video cambiata; bitrate aggiornato |

### C4. Email — 5 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `email.bounced` | `track` / `Email Bounced` | Email non consegnata | A | No | Il server segnala che il messaggio non può essere consegnato. Non prova che l’indirizzo sia inesistente: distinguere cause transitorie e permanenti se il provider le espone. | email rimbalzata; errore consegna email |
| `email.delivered` | `track` / `Email Delivered` | Email consegnata al server | A | No | Il server destinatario conferma ricezione del messaggio. Non garantisce collocazione in posta in arrivo o lettura umana. | email consegnata; messaggio accettato |
| `email.link_clicked` | `track` / `Email Link Clicked` | Link email selezionato | A | No | Rilevato clic su collegamento nel corpo email. Il segnale non dimostra da solo lettura completa o acquisto. | clic in email; link newsletter cliccato |
| `email.marked_as_spam` | `track` / `Email Marked as Spam` | Email segnalata come spam | A | No | Il destinatario marca il messaggio come spam. Non equivale automaticamente a una disiscrizione globale o a un bounce. | segnalazione spam; messaggio marcato spam |
| `email.opened` | `track` / `Email Opened` | Apertura email rilevata | A | No | Il sistema email registra un’apertura. Non prova lettura completa o azione umana; conservare il metodo di rilevazione del provider. | email aperta; apertura newsletter |

### C5. Live Chat — 4 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `chat.conversation_ended` | `track` / `Live Chat Conversation Ended` | Conversazione chat conclusa | A | No | Cliente o operatore dichiara esplicitamente conclusa la conversazione. Non inferire conclusione dalla sola inattività o chiusura della pagina. | chat conclusa; conversazione terminata |
| `chat.conversation_started` | `track` / `Live Chat Conversation Started` | Conversazione chat iniziata | A | No | Il cliente invia il primo messaggio della conversazione. Non basta aprire il widget senza inviare un messaggio. | chat iniziata; nuova conversazione assistenza |
| `chat.message_received` | `track` / `Live Chat Message Received` | Messaggio chat ricevuto dal cliente | A | No | Il cliente riceve un messaggio dall’operatore. La direzione è operatore verso cliente; non prova lettura. | risposta ricevuta in chat; messaggio dal supporto |
| `chat.message_sent` | `track` / `Live Chat Message Sent` | Messaggio chat inviato dal cliente | A | No | Il cliente invia un messaggio all’operatore. La direzione è cliente verso operatore. | ha scritto in chat; messaggio al supporto |

### C6. Experimentation — 1 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `experiment.viewed` | `track` / `Experiment Viewed` | Variante di esperimento visualizzata | A | No | È stata registrata l’esposizione a una variante di un esperimento attivo. Non indica conversione, risultato dell’esperimento o prova del suo effetto causale. | variante vista; esposizione a test A/B |

### C7. Advanced Leads — 2 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `lead.demo_requested` | `track` / `Demo Requested` | Dimostrazione richiesta | C | No | È stata registrata una richiesta esplicita di dimostrazione del prodotto. Non prova appuntamento confermato o demo svolta. | richiesta demo; ha chiesto una dimostrazione |
| `lead.qualified` | `track` / `Lead Qualified` | Contatto qualificato | C | No | Il sistema commerciale ha classificato il contatto come qualificato secondo una regola dichiarata. Non equivale a semplice acquisizione di un contatto o a vendita. | lead qualificato; contatto qualificato |

### C8. Subscription — 4 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `subscription.cancelled` | `track` / `Subscription Cancelled` | Abbonamento cancellato | C | No | È diventata effettiva la cancellazione dell’abbonamento secondo il contratto proposto. Una cancellazione programmata per fine periodo deve avere un evento o stato distinto. Non è una disiscrizione dalle email. | abbonamento terminato per cancellazione; cancellazione abbonamento effettiva |
| `subscription.renewed` | `track` / `Subscription Renewed` | Abbonamento rinnovato | C | No | Il sistema conferma l’avvio di un nuovo periodo di un abbonamento già esistente. Non basta una fattura emessa o un tentativo di pagamento; precisare se il rinnovo è subordinato all’incasso. | rinnovo abbonamento; abbonamento rinnovato |
| `subscription.started` | `track` / `Subscription Started` | Abbonamento iniziato | B | No | È stato registrato l’inizio di un abbonamento al servizio secondo il contratto locale. Non basta una newsletter o una prova; il nome di esempio non garantisce che sia stato incassato un pagamento. | abbonamento attivato; sottoscrizione iniziata |
| `subscription.upgraded` | `track` / `Subscription Upgraded` | Abbonamento passato a un piano superiore | B | No | È stato registrato il passaggio dell’abbonamento a un piano superiore. Il nome di esempio non definisce calcolo del conguaglio o incasso; non rappresenta ogni modifica di piano. | upgrade abbonamento; passaggio a piano superiore |

### C9. Technical — 2 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `identity.alias_recorded` | `alias` (qualsiasi `name`) | Associazione tra identificatori richiesta | A | No | È stata ricevuta una chiamata alias con un identificatore precedente e uno corrente. È un’operazione sull’identità per i sistemi che la supportano; non è un sinonimo di evento né prova universale di fusione di profili. | alias identità; associazione identificatori |
| `app.screen_viewed` | `screen` (qualsiasi `name`) | Schermata visualizzata | A | No | È stata registrata la visualizzazione di una schermata applicativa. Il nome può essere assente nella specifica generale. Non certifica una sessione distinta o un’azione di business. | schermata vista; visita schermata app |

### D. Legacy Technical — 1 definizioni

| ID semantico | Matcher canonico | Display name | Origine | Preset | Descrizione e limiti | Sinonimi |
|---|---|---|:---:|:---:|---|---|
| `legacy.identity_merge` | `merge` (qualsiasi `name`) | Richiesta storica di unione identità | D | No | Payload storico merge di RudderStack per risoluzione identità in warehouse. Serve solo a riconoscere archivi di implementazioni precedenti; non è un metodo corrente né un’azione del cliente. | unione identità storica; merge storico |

## Appendice B — Tutte le 16 varianti tecniche

**Tutte sono disabilitate e richiedono validazione amministrativa.** La colonna «Relazione» descrive la classe documentata nel registro, **non** certifica un'equivalenza valida per ogni installazione.

| ID variante | `track.event` alternativo | Destinazione semantica | Relazione | Ambito dichiarato | Attivazione |
|---|---|---|---|---|---|
| `compat.segment.viewed_product_category` | `Viewed Product Category` | `commerce.product_list_viewed` | `documented_legacy_alias` | `v1` | Disabilitata; conferma necessaria |
| `compat.segment.viewed_promotion` | `Viewed Promotion` | `commerce.promotion_viewed` | `documented_legacy_alias` | `documented_alias_version_unspecified` | Disabilitata; conferma necessaria |
| `compat.segment.clicked_promotion` | `Clicked Promotion` | `commerce.promotion_clicked` | `documented_legacy_alias` | `documented_alias_version_unspecified` | Disabilitata; conferma necessaria |
| `compat.segment.clicked_product` | `Clicked Product` | `commerce.product_clicked` | `documented_legacy_alias` | `v1_retrospective` | Disabilitata; conferma necessaria |
| `compat.segment.viewed_product` | `Viewed Product` | `commerce.product_viewed` | `documented_legacy_alias` | `v1` | Disabilitata; conferma necessaria |
| `compat.segment.added_product` | `Added Product` | `commerce.product_added` | `documented_legacy_alias` | `v1` | Disabilitata; conferma necessaria |
| `compat.segment.removed_product` | `Removed Product` | `commerce.product_removed` | `documented_legacy_alias` | `v1` | Disabilitata; conferma necessaria |
| `compat.segment.started_order` | `Started Order` | `commerce.checkout_started` | `documented_legacy_alias` | `ga_enhanced_ecommerce` | Disabilitata; conferma necessaria |
| `compat.segment.viewed_checkout_step` | `Viewed Checkout Step` | `commerce.checkout_step_viewed` | `documented_legacy_alias` | `ga_enhanced_ecommerce` | Disabilitata; conferma necessaria |
| `compat.segment.completed_checkout_step` | `Completed Checkout Step` | `commerce.checkout_step_completed` | `documented_legacy_alias` | `ga_enhanced_ecommerce` | Disabilitata; conferma necessaria |
| `compat.segment.updated_order` | `Updated Order` | `commerce.order_updated` | `documented_legacy_alias` | `ga_enhanced_ecommerce` | Disabilitata; conferma necessaria |
| `compat.segment.completed_order` | `Completed Order` | `commerce.order_completed` | `documented_legacy_alias` | `v1_retrospective` | Disabilitata; conferma necessaria |
| `compat.segment.user_signed_up` | `User Signed Up` | `auth.signed_up` | `documented_naming_variant` | `segment_b2b_saas` | Disabilitata; conferma necessaria |
| `compat.segment.user_signed_in` | `User Signed In` | `auth.signed_in` | `documented_naming_variant` | `segment_b2b_saas` | Disabilitata; conferma necessaria |
| `compat.segment.user_signed_out` | `User Signed Out` | `auth.signed_out` | `documented_naming_variant` | `segment_b2b_saas` | Disabilitata; conferma necessaria |
| `compat.customer.purchase_completed` | `Purchase Completed` | `commerce.order_completed` | `customer_configurable_equivalence` | `destination_example_name_only` | Disabilitata; conferma necessaria |

## Appendice C — Fonti di riferimento per le convenzioni e le tassonomie

Le attribuzioni **per singola definizione** si trovano in `standard`, `sources` e `source_url` nel JSON. Questi sono i riferimenti principali usati dalla ricerca originaria (verificata il 29 ottobre 2026):

- [Segment — Track](https://www.twilio.com/docs/segment/connections/spec/track)
- [Segment — Page](https://www.twilio.com/docs/segment/connections/spec/page)
- [Segment — Identify](https://www.twilio.com/docs/segment/connections/spec/identify)
- [Segment — Group](https://www.twilio.com/docs/segment/connections/spec/group)
- [Segment — Ecommerce V2](https://www.twilio.com/docs/segment/connections/spec/ecommerce/v2)
- [Segment — B2B SaaS](https://www.twilio.com/docs/segment/connections/spec/b2b-saas)
- [Segment — Data Collection Best Practices](https://www.twilio.com/docs/segment/protocols/tracking-plan/best-practices)
- [Segment — Video](https://www.twilio.com/docs/segment/connections/spec/video)
- [Segment — Email](https://www.twilio.com/docs/segment/connections/spec/email)
- [RudderStack — Standard Track](https://www.rudderstack.com/docs/event-spec/standard-events/track/)
- [RudderStack — Ecommerce](https://www.rudderstack.com/docs/event-spec/ecommerce-events-spec/)
- [RudderStack — Ordering](https://www.rudderstack.com/docs/event-spec/ecommerce-events-spec/ordering/)
- [RudderStack — Identify](https://www.rudderstack.com/docs/event-spec/standard-events/identify/)

**Conclusione operativa:** il risultato desiderato è un catalogo di eventi semantici affidabile e modificabile per workspace che risolva correttamente i messaggi `type`/`event`/`name`, conservi nomi e provenienze documentati e non introduca equivalenze tecniche o semantiche arbitrarie. Il modo di esporlo tramite MCP non fa parte di questa specifica.
