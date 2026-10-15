// Wait for both download and decoding, including images in unopened home panels.
// Failed assets settle too: an unavailable image must not trap the fan on the splash.
export async function preloadFanHomeImages(urls, { signal, ImageClass = globalThis.Image } = {}) {
  if (!ImageClass || signal?.aborted) return;
  await Promise.allSettled([...new Set(urls.filter(Boolean))].map((url) => new Promise((resolve) => {
    const image = new ImageClass();
    const finish = () => {
      image.onload = null;
      image.onerror = null;
      signal?.removeEventListener('abort', finish);
      resolve();
    };
    image.onload = async () => {
      try { await image.decode?.(); } catch { /* unsupported or invalid image */ }
      finish();
    };
    image.onerror = finish;
    signal?.addEventListener('abort', finish, { once: true });
    image.src = url;
  })));
}
