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

	function expectTraits(kind, traits) {
		assertEquals(storage.getTraits(kind), traits);
	}

	function expectUserID(id) {
		assertEquals(storage.getUserID(), id);
	}

	expectAnonymousID(null);
	expectGroupID(null);
	expectSession(null, 0, false);
	expectTraits('user', {});
	expectTraits('group', {});
	expectUserID(null);

	storage.setAnonymousID('703a1h3b830');
	expectAnonymousID('703a1h3b830');

	storage.setGroupID('72047285');
	expectGroupID('72047285');
	storage.setGroupID();
	expectGroupID(null);

	storage.setSession();
	expectSession(null, 0, false);

	storage.setSession(1706175160340, 1706176628710, false);
	expectSession(1706175160340, 1706176628710, false);

	storage.setSession(1706178514540, 1706178239698, true);
	expectSession(1706178514540, 1706178239698, true);

	storage.setTraits('user', { name: 'John' });
	expectTraits('user', { name: 'John' });
	storage.setTraits('user', { name: 0n });
	expectTraits('user', { name: 'John' });
	storage.setTraits('user', {});
	expectTraits('user', {});
	storage.setTraits('user', { name: 'John' });
	storage.setTraits('user');
	expectTraits('user', {});

	storage.setTraits('group', { name: 'Acme' });
	expectTraits('group', { name: 'Acme' });
	storage.setTraits('group', { name: 0n });
	expectTraits('group', { name: 'Acme' });
	storage.setTraits('group', {});
	expectTraits('group', {});
	storage.setTraits('group', { name: 'Acme' });
	storage.setTraits('group');
	expectTraits('group', {});

	storage.setUserID('86103517');
	expectUserID('86103517');
	storage.setUserID();
	expectUserID(null);

	storage.setSession();
	expectSession(null, 0, false);
});
