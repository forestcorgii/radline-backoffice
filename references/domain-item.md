# Domain — Item, Brand, Category, UomSetting

> Back to [[domain-overview]] · Related: [[database-schema]]

## Brand Entity
**File:** [brand.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/brand.go)

```go
type Brand struct {
    ID   int
    Code string
    Name string
}
```

**Invariants (enforced by `NewBrand`):**
- `Code` cannot be empty
- `Name` cannot be empty

---

## Category Entity
**File:** [category.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/category.go)

```go
type Category struct {
    ID   int
    Code string
    Name string
}
```

**Invariants (enforced by `NewCategory`):**
- `Code` cannot be empty
- `Name` cannot be empty

---

## Item Entity
**File:** [item.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/item.go)

```go
type Item struct {
    ID          int
    Code        string
    Description string
    DefaultUOM  string
    Model       string
    BrandID     int
    CategoryID  int
    Variation   string
    Remarks     string
}
```

**Invariants (enforced by `NewItem`):**
- `Code` cannot be empty
- `Description` cannot be empty
- `DefaultUOM` cannot be empty

**Note:** `BrandID` and `CategoryID` are `int` in the domain (0 = unset) but `*int` (nullable) in the [[models-layer|models DTO]]. The handler layer handles this mapping.

---

## UomSetting Value Object
**File:** [uom_setting.go](file:///c:/Users/USER/Documents/Coding%20Projects/antigravity/radline/domain/uom_setting.go)

```go
type UomSetting struct {
    MUOM             string
    ConversionFactor float64
}
```

**Invariants (enforced by `NewUomSetting`):**
- `MUOM` cannot be empty
- `ConversionFactor` must be > 0

**Usage:** Maps a non-default UOM to a multiplier relative to the item's `DefaultUOM`. For example, if `DefaultUOM = "PCS"` and `MUOM = "BOX"` with `ConversionFactor = 10`, then 1 BOX = 10 PCS. See [[domain-stock]] for how this is used in stock calculations.

## Related
- [[domain-stock]] — How items participate in stock calculations
- [[handlers-master-data]] — CRUD handlers for these entities
- [[database-schema]] — SQLite table definitions
- [[domain-testing]] — Validation test cases
