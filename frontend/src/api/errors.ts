/**
 * isNetworkError distinguishes a connectivity failure (which should fall back to
 * the offline queue / cache) from an application error such as validation or
 * conflict (which should be re-thrown). ApiError carries a numeric HTTP
 * `status`; a bare fetch failure (TypeError) does not.
 */
export function isNetworkError(err: unknown): boolean {
  if (err instanceof TypeError) return true;
  return !(typeof err === 'object' && err !== null && 'status' in err);
}
