/** Aggregate cursor pages for existing selectors/tables without silently truncating their data. */
export async function collectPages<T>(
  fetchPage: (cursor?: string) => Promise<{ items: T[]; nextCursor?: string }>,
): Promise<T[]> {
  const items: T[] = [];
  const seen = new Set<string>();
  let cursor: string | undefined;
  do {
    const page = await fetchPage(cursor);
    items.push(...page.items);
    cursor = page.nextCursor || undefined;
    if (cursor && seen.has(cursor))
      throw new Error("The server repeated a pagination cursor.");
    if (cursor) seen.add(cursor);
  } while (cursor);
  return items;
}
