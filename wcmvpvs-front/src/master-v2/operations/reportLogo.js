const MAX_SOURCE_BYTES = 4 * 1024 * 1024;
const MAX_ENCODED_LENGTH = 512 * 1024;
const MAX_DIMENSION = 2048;

export async function prepareReportLogo(file, signal) {
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    throw new Error('Seleziona un logo PNG, JPEG o WebP.');
  }
  if (file.size > MAX_SOURCE_BYTES) throw new Error('Il file originale deve pesare al massimo 4 MB.');
  const url = URL.createObjectURL(file);
  try {
    const image = new Image();
    image.src = url;
    try {
      await image.decode();
    } catch {
      throw new Error('Il file non contiene un’immagine valida.');
    }
    signal.throwIfAborted();
    let { naturalWidth: width, naturalHeight: height } = image;
    if (!width || !height || Math.max(width, height) > MAX_DIMENSION) {
      throw new Error('Il logo deve avere da 1 a 2048 pixel per lato.');
    }

    const canvas = document.createElement('canvas');
    const context = canvas.getContext('2d');
    if (!context) throw new Error('Il browser non può elaborare l’immagine.');
    // The report API accepts PNG and JPEG. PNG preserves WebP transparency.
    const outputType = file.type === 'image/jpeg' ? 'image/jpeg' : 'image/png';
    for (let attempt = 0; attempt < 20; attempt += 1) {
      canvas.width = width;
      canvas.height = height;
      context.drawImage(image, 0, 0, width, height);
      const data = canvas.toDataURL(outputType, 0.9);
      if (data.length <= MAX_ENCODED_LENGTH && /^data:image\/(png|jpeg);base64,/.test(data)) {
        return { data, width, height, kilobytes: Math.ceil(data.length / 1024) };
      }
      width = Math.max(1, Math.round(width * 0.8));
      height = Math.max(1, Math.round(height * 0.8));
    }
    throw new Error('Il logo supera il limite di 512 KB. Usa un’immagine più semplice.');
  } finally {
    URL.revokeObjectURL(url);
  }
}
