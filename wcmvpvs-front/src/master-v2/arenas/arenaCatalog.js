import { masterApi } from '../api';

export const ARENAS_ENDPOINT = '/admin/master/arenas';

export async function readArenaCatalog(config) {
  const { data } = await masterApi.get(ARENAS_ENDPOINT, config);
  return Array.isArray(data?.arenas) ? data.arenas : [];
}
