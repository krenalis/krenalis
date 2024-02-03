import { assertEquals } from 'https://deno.land/std@0.212.0/assert/mod.ts';
import Storage from './storage.js';

Deno.test('Storage', () => {
	localStorage.clear();

	const storage = new Storage();

	function expectAnonymousId(id) {
		assertEquals(storage.anonymousId(), id);
	}

	function expectGroupId(id) {
		assertEquals(storage.groupId(), id);
	}

	function expectSession(id, expiration, start) {
		const [actualId, actualExpiration, actualStart] = storage.session();
		assertEquals(actualId, id);
		assertEquals(actualExpiration, expiration);
		assertEquals(actualStart, start);
	}

	function expectTraits(kind, traits) {
		assertEquals(storage.traits(kind), traits);
	}

	function expectUserId(id) {
		assertEquals(storage.userId(), id);
	}

	expectAnonymousId(null);
	expectGroupId(null);
	expectSession(null, 0, false);
	expectTraits('user', {});
	expectTraits('group', {});
	expectUserId(null);

	storage.setAnonymousId('703a1h3b830');
	expectAnonymousId('703a1h3b830');

	storage.setGroupId('72047285');
	expectGroupId('72047285');
	storage.setGroupId();
	expectGroupId(null);

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

	storage.setUserId('86103517');
	expectUserId('86103517');
	storage.setUserId();
	expectUserId(null);

	storage.setSession();
	expectSession(null, 0, false);
});
