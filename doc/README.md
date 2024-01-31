# Documentation

To navigate the documentation, download the latest release of mdBook

https://github.com/rust-lang/mdBook/releases

and start the mdbook CLI:

>  mdbook serve doc --open

To ignore generated files from Git, add the following lines to ".gitignore":

> /doc/book/*.html
> /doc/book/images/
> /doc/book/searchindex.js
> /doc/book/searchindex.json