import { assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import Storage from './storage.js';

Deno.test('Storage', () => {
	localStorage.clear();

	const storage = new Storage();

	function expectAnonymousID(id) {
		assertEquals(storage.getAnonymousID(), id);
	}

	function expectGroupID(id) {
		assertEquals(storage.getGroupID(), id);
	}

	function expectSession(id, expiration, start) {
		const [actualID, actualExpiration, actualStart] = storage.getSession();
		assertEquals(actualID, id);
		assertEquals(actualExpiration, expiration);
		assertEquals(actualStart, start);
	}

	function expectTraits(traits) {
		assertEquals(storage.getTraits(), traits);
	}

	function expectUserID(id) {
		assertEquals(storage.getUserID(), id);
	}

	expectAnonymousID(null);
	expectGroupID(null);
	expectSession(null, 0, false);
	expectTraits({});
	expectUserID(null);

	storage.setAnonymousID('703a1h3b830');
	expectAnonymousID('703a1h3b830');

	storage.setGroupID('72047285');
	expectGroupID('72047285');
	storage.setGroupID(null);
	expectGroupID(null);

	storage.setSession(null, 0, false);
	expectSession(null, 0, false);

	storage.setSession(1706175160340, 1706176628710, false);
	expectSession(1706175160340, 1706176628710, false);

	storage.setSession(1706178514540, 1706178239698, true);
	expectSession(1706178514540, 1706178239698, true);

	storage.setTraits({ name: 'John' });
	expectTraits({ name: 'John' });
	storage.setTraits({ name: 0n });
	expectTraits({ name: 'John' });
	storage.setTraits({});
	expectTraits({});
	storage.setTraits({ name: 'John' });
	storage.setTraits(null);
	expectTraits({});

	storage.setUserID('86103517');
	expectUserID('86103517');
	storage.setUserID(null);
	expectUserID(null);

	storage.setSession(null);
	expectSession(null, 0, false);
});
