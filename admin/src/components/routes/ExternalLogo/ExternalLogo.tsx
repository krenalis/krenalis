import React, { useContext, useEffect, useState } from 'react';
import './ExternalLogo.css';
import AppContext from '../../../context/AppContext';
import UnknownLogo from '../../base/UnknownLogo/UnknownLogo';

interface ExternalLogoProps {
	code: string | null;
	path: string;
	slot?: string;
}

const okSrcCache = new Map<string, string | null>();

const ExternalLogo = ({ code, path, slot }: ExternalLogoProps) => {
	const [src, setSrc] = useState<string | null>(null);

	const { publicMetadata } = useContext(AppContext);

	useEffect(() => {
		setSrc(null);

		let sources = [];
		if (code != null) {
			sources = publicMetadata.externalAssetsURLs.map((url: string) => {
				let source = url + 'admin/';
				if (path !== '') {
					source += `${path}/`;
				}
				source += `${code}.svg`;
				return source;
			});
		}

		if (!code || sources.length === 0) {
			return;
		}

		const cached = okSrcCache.get(code);
		if (cached !== undefined) {
			setSrc(cached);
			return;
		}

		let cancelled = false;

		(async () => {
			for (const src of sources) {
				try {
					// test the images outside of the DOM to prevent delays and
					// flickers during render.
					const ok = await isImageOK(src);
					if (cancelled) {
						return;
					}
					if (ok) {
						okSrcCache.set(code, src);
						setSrc(src);
						return;
					}
				} catch {
					// try the next image.
				}
			}
			// all images have failed.
			if (!cancelled) {
				okSrcCache.set(code, null);
				setSrc(null);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [code, publicMetadata.externalAssetsURLs]);

	if (src != null) {
		return <img src={src} slot={slot != null ? slot : undefined} className='external-logo__image' />;
	}
	return <UnknownLogo size={21} />;
};

async function isImageOK(src: string): Promise<boolean> {
	const img = new Image();
	img.loading = 'eager';
	const isOK: boolean = await new Promise((resolve) => {
		img.onload = () => resolve(true);
		img.onerror = () => resolve(false);
		img.src = src;
	});
	if (!isOK) {
		return false;
	}
	return true;
}

export { ExternalLogo };
