export default async function call(url, json) {
    let body;
    if (json) body = JSON.stringify(json);

    let res;
    try {
        res = body ? await fetch(url, { method: 'POST', body: body }) : await fetch(url);
    } catch (err) {
        return [null, `error while fetching ${url}: ${err}`];
    }

    if (res.status !== 200) {
        let error;
        switch (res.status) {
            case 500:
                error = 'Internal Server Error';
                break;
            case 400:
                error = 'Bad Request';
                break;
            default:
                error = "Unknown Server Error";
                break;
        }
        let errorDescription = res.headers.get('x-error');
        if (errorDescription) error += `: ${errorDescription}`;
        return [null, error];
    }


    let data;
    try {
        data = await res.json();
    } catch (err) {
        return [null, `error while parsing json response from ${url}: ${err}`];
    }

    return [data, null];
}