# Spec: Team Knowledge Base Search

## Objective

Build a search feature for an internal team knowledge base. Users should be able to search published documents by keyword and see ranked results with title, snippet, and source path.

## Tech Stack

- Runtime: Node.js
- UI: React
- Data source: existing Markdown documents under `content/`
- Search index: in-memory index for the first version

## Commands

- Build: `npm run build`
- Test: `npm test`
- Lint: `npm run lint`

## Project Structure

- `src/search/`: indexing and ranking logic
- `src/components/SearchBox.tsx`: search input UI
- `src/components/SearchResults.tsx`: result list UI
- `tests/search/`: unit tests for indexing, tokenization, and ranking

## Code Style

- Keep search logic pure and testable.
- Prefer small functions with explicit inputs and outputs.
- Do not add external search dependencies in the first version.

## Testing Strategy

- Unit-test tokenization, indexing, and ranking.
- Component-test empty, loading, and populated result states.
- Include regression tests for case-insensitive search and missing snippets.

## Boundaries

- Always: preserve existing document paths in results.
- Ask first: adding an external search service or database.
- Never: index unpublished or private documents outside `content/`.

## Success Criteria

- A user can search a keyword and receive relevant ranked results.
- Results include title, snippet, and source path.
- Search handles empty query, no results, and case-insensitive matches.
- Tests cover indexing and ranking behavior.
