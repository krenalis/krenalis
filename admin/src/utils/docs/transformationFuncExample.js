export let transformationFuncExample = `func(input map[string]any, timestamps map[string]time.Time) (map[string]any, map[string]time.Time, error) {
    out := map[string]any{}
    outTimestamps := map[string]time.Time{}
    if firstName, ok := input["firstname"]; ok {
        out["FirstName"] = firstName
        outTimestamps["FirstName"] = timestamps["firstname"]
    }
    if lastName, ok := input["lastname"]; ok {
        out["LastName"] = lastName
        outTimestamps["LastName"] = timestamps["lastname"]
    }
    if email, ok := input["email"]; ok {
        out["Email"] = email
        outTimestamps["Email"] = timestamps["email"]
    }
    return out, outTimestamps, nil
}`
