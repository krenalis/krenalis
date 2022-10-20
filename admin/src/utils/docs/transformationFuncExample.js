export let transformationFuncExample = `func(input map[string]any) (map[string]any, error) {
    out := map[string]any{}
    if firstName, ok := input["firstname"]; ok {
        out["FirstName"] = firstName
    }
    if lastName, ok := input["lastname"]; ok {
        out["LastName"] = lastName
    }
    if email, ok := input["email"]; ok {
        out["Email"] = email
    }
    return out, nil
}`
