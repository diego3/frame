### Techniques to manage technical debt

Ways to handle it without changing your code immediately:

- **Backlog + prioritization**: Keep a list (e.g. in `TECH_DEBT.md` or TDRs) and assign priority (e.g. P0–P2) and maybe impact/effort so you can pick what to do next.
- **Time-boxing**: Reserve a fixed slice of each sprint or release for debt (e.g. “20% of capacity” or “N points per sprint”) and choose items from the backlog.
- **Boy Scout Rule**: When touching a file (e.g. `mainmenu.go`), fix or reduce one small piece of debt there (e.g. move one button to data or add a note in the backlog) so debt doesn’t grow.
- **Link debt to ADRs**: For each debt item, note which ADR or decision it violates (e.g. “ADR-001: data-driven UI”). That keeps “what we decided” and “what we haven’t done yet” aligned.
- **Refactor when adding features**: When you need to change the main menu anyway (e.g. new buttons or screens), do “mount UI from file” as part of that work instead of adding more hardcoded UI.
- **Automate discovery**: Use grep/scripts or CI to find `TODO` / `FIXME` / `HACK` in the repo and feed that into the backlog so nothing stays invisible.