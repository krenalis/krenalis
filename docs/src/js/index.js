import { handleSidebar } from "./sidebar.js";
import { buildTableOfContent } from "./table-of-content.js";

const setup = () => {
    handleSidebar();
    buildTableOfContent();
};

setup();

(() => {
	const RX = /^\s*(\d+)\.\s+/;

	// Find the next numbered H3 not yet processed.
	const nextNumbered = () => {
		const h3s = document.querySelectorAll('h3');
		for (const h of h3s) {
			if (!h.closest('.steps') && RX.test(h.textContent)) return h;
		}
		return null;
	};

	let first;
	while ((first = nextNumbered())) {
		const parent = first.parentNode;
		const steps = document.createElement('div');
		steps.className = 'steps';
		parent.insertBefore(steps, first);

		// Move sibling nodes until the next H2/non-consecutive H3 or the end of the parent.
		let n = first;
		const firstMatch = first.textContent.match(RX);
		let expected = firstMatch ? parseInt(firstMatch[1], 10) + 1 : null;
		while (n && n.parentNode === parent) {
			if (n.tagName === 'H2') break;
			if (n !== first && n.tagName === 'H3') {
				const match = n.textContent.match(RX);
				const hasNumber = match && match[1];
				const current = hasNumber ? parseInt(match[1], 10) : null;
				if (!hasNumber || (expected !== null && current !== expected)) break;
				expected = current !== null ? current + 1 : null;
			}
			const next = n.nextSibling;
			steps.appendChild(n);
			n = next;
		}

		// Transform the numbered H3s of the block.
		steps.querySelectorAll('h3').forEach(h => {
			const m = h.textContent.match(RX);
			if (!m) return;
			const num = m[1];

			h.classList.add('step-heading');
			// remove "N. " from the first text node.
			const tn = (h.firstChild && h.firstChild.nodeType === Node.TEXT_NODE) ? h.firstChild : null;
			if (tn) tn.nodeValue = tn.nodeValue.replace(RX, '');
			else h.innerHTML = h.innerHTML.replace(RX, '');

			// numeric badge.
			if (!h.querySelector('.step')) {
				const badge = document.createElement('span');
				badge.className = 'step';
				badge.setAttribute('aria-hidden', 'true');
				badge.textContent = num;
				h.insertBefore(badge, h.firstChild);
			}
		});
	}
})();