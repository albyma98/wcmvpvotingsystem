export function portalErrorCode(error) {
  const data = error?.response?.data
  if (typeof data !== 'string') return data?.error
  try {
    return JSON.parse(data)?.error
  } catch {
    return undefined
  }
}
