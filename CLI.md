# CLI

pkgs comes with a command-line utility in the `cli` folder.

Resources are addressed with "resource paths" that begin with `/` (and sometimes end with `/`, depending on what you're doing).

## List package members

You can list the members of a package with `ul ls [resource]`.

```
% ul ls /
Files          URI                                                                    Created                   Modified                  Format              Size
context.jsonld dweb:/ipfs/bafkreibwyzeetse6wpzntw2rj5jkblxccysntihmezupsjwgx532ttrwxm 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 221
package.jsonld dweb:/ipfs/bafkreif7y43zk6p7hlvmarjveblgiqyy2266wxounchapnakmuxdjqdpam 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 1240
schema.jsonld  dweb:/ipfs/bafkreigpvc6gujkuvja4usyjykcpakdmjtoegehls5rdln4mzxenvgxzgq 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 715
shex.jsonld    dweb:/ipfs/bafkreicqpmn3cyvyr4ioy6link2deaik53nih4tqobee7xmco2xpfiu7gu 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 30939

```

## Make a new package

You can make a new package with `mkpkg [resource]`. The path has to be an immediate child of an existing package, and no resource at that path can already exist.

```
% ul mkpkg /foo
% ul ls /
Packages       URI
foo            ul:bafkreibql4lpadeg43gfrdtsgt4d7zgwqnk6kkavviofsn2vh5xpb6akuq#c14n0

Files          URI                                                                    Created                   Modified                  Format              Size
context.jsonld dweb:/ipfs/bafkreibwyzeetse6wpzntw2rj5jkblxccysntihmezupsjwgx532ttrwxm 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 221
package.jsonld dweb:/ipfs/bafkreif7y43zk6p7hlvmarjveblgiqyy2266wxounchapnakmuxdjqdpam 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 1240
schema.jsonld  dweb:/ipfs/bafkreigpvc6gujkuvja4usyjykcpakdmjtoegehls5rdln4mzxenvgxzgq 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 715
shex.jsonld    dweb:/ipfs/bafkreicqpmn3cyvyr4ioy6link2deaik53nih4tqobee7xmco2xpfiu7gu 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 30939

% ul ls /foo
% ul mkpkg /foo/bar
% ul ls /foo
Packages URI
bar      ul:bafkreidgidyuetiueeornvlhd7jg6lp4w5c2t647jfcevdhujkwd7h22ge#c14n0

% ul ls /foo/bar
```

## Get a resource

The command `ul get --format [format] [resource]` fetches a representation of the given resource in the given format. For packages and assertions, `--format` must be one of `application/n-quads` (N-Quads), `application/ld+json` (JSON-LD), or `application/json` (RDFJS), or it will default to `application/n-quads` if not given. For files, the `--format` flag is ignored, since files only have one representation.

```
% ul get /context.jsonld
{
	"@context": {
		"xsd": "http://www.w3.org/2001/XMLSchema#",
		"dcterms": "http://purl.org/dc/terms/",
		"prov": "http://www.w3.org/ns/prov#",
		"ldp": "http://www.w3.org/ns/ldp#",
		"schema": "http://schema.org/"
	}
}

% ul get /foo/bar
<dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354> <http://purl.org/dc/terms/extent> "4"^^<http://www.w3.org/2001/XMLSchema#integer> .
_:c14n0 <http://purl.org/dc/terms/created> "2020-05-05T19:10:44-04:00"^^<http://www.w3.org/2001/XMLSchema#dateTime> .
_:c14n0 <http://purl.org/dc/terms/modified> "2020-05-05T19:10:44-04:00"^^<http://www.w3.org/2001/XMLSchema#dateTime> .
_:c14n0 <http://purl.org/dc/terms/title> "bar" .
_:c14n0 <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://www.w3.org/ns/ldp#DirectContainer> .
_:c14n0 <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://www.w3.org/ns/prov#Collection> .
_:c14n0 <http://www.w3.org/ns/ldp#hasMemberRelation> <http://www.w3.org/ns/prov#hadMember> .
_:c14n0 <http://www.w3.org/ns/ldp#membershipResource> <dweb:/ipns/QmdwEbzyeYHeeq7ky81hdFZDJ6FCeyuYJdV8WjhttEn4dU/foo/bar> .
_:c14n0 <http://www.w3.org/ns/prov#value> <dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354> .
% ul get --format application/ld+json /foo/bar
{
  "@context": "dweb:/ipfs/bafkreif7y43zk6p7hlvmarjveblgiqyy2266wxounchapnakmuxdjqdpam",
  "@type": [
    "ldp:DirectContainer",
    "prov:Collection"
  ],
  "created": "2020-05-05T19:10:44-04:00",
  "ldp:hasMemberRelation": {
    "@id": "prov:hadMember"
  },
  "members": {},
  "modified": "2020-05-05T19:10:44-04:00",
  "resource": "dweb:/ipns/QmdwEbzyeYHeeq7ky81hdFZDJ6FCeyuYJdV8WjhttEn4dU/foo/bar",
  "title": "bar",
  "value": {
    "extent": 4,
    "id": "dweb:/ipfs/bafybeiczsscdsbs7ffqz55asqdf3smv6klcw3gofszvwlyarci47bgf354"
  }
}
```

## Put a named resource

Package members might be named, or they might be unnamed. Packages are always named, but assertions and files might only be identified by their hash.

The idea is to give curators flexibility: you shouldn't have to name everything - you should be able to just quickly insert an anonymous assertion - but if you have specific files or assertions that you expect to "edit", it makes sense to give them a persistent identity.

In a package `/foo`, an assertion "named" `bar` can be addressed as `/foo/bar`. Similarly, a file named `baz.txt` can be addressed as `/foo/baz.txt`

When naming assertions, you _shouldn't_ put a file extension in the name, because the assertion has many different serialized representations. But when naming a file, you _probably should_ put a file extension in the name, if you know a good one.

To create and update named resources, use the `ul put` command. You have to explicitly say what kind of resource you're putting by using exactly one of the `--package`, `--assertion`, or `--file` boolean flags. And you need to specify the representation of the file you're putting using `--format` - this is required for files, and will default to `application/n-quads` for packages and assertions.

```
% echo 'Hello World!' > hello.txt
% ul put --file --format text/plain hello.txt /foo/.
% ul ls /foo
Packages  URI
bar       ul:bafkreidgidyuetiueeornvlhd7jg6lp4w5c2t647jfcevdhujkwd7h22ge#c14n0

Files     URI                                                                    Created                   Modified                  Format     Size
hello.txt dweb:/ipfs/bafkreiadxiqe4ugre3sgotaalycnqlueyijwm6ak6h2dxvkkg6aww2vtia 2020-05-05T19:34:35-04:00 2020-05-05T19:34:35-04:00 text/plain 13

% echo '_:b0 <http://schema.org/name> "John Doe" .' > john-doe
% ul put --assertion john-doe /foo/bar/jd
% ul ls /foo/bar
Assertions URI                                                            Created                   Modified
jd         ul:bafkreigsyouvprcm5wqo7l5zeehitmkiw25gjvrbz5d4pqeowaupw3zzdi 2020-05-05T19:40:06-04:00 2020-05-05T19:40:06-04:00

```

## Post an unnamed resource

So what if we want to add an assertion or file without giving it a name? For this we use the `post` command and we add a trailing slash to resource path of the package we want to add to. Just like `put`, we need to explicitly say what kind of resource we're adding, although here we're limited to `--assertion` and `--file` since all packages are named. And `--format` works the same way - require for files, and defaults to `application/n-quads` for assertions.

```
% ul post --file hello.txt /foo/bar/
% ul ls /foo/bar
Assertions URI                                                                    Created                   Modified
jd         ul:bafkreigsyouvprcm5wqo7l5zeehitmkiw25gjvrbz5d4pqeowaupw3zzdi         2020-05-05T19:40:06-04:00 2020-05-05T19:40:06-04:00

Files      URI                                                                    Created                   Modified                  Format     Size
           dweb:/ipfs/bafkreiadxiqe4ugre3sgotaalycnqlueyijwm6ak6h2dxvkkg6aww2vtia 2020-05-05T19:45:21-04:00                           text/plain 13

```

Now the file is just "there" - known only as `dweb:/ipfs/bafkreiadxiqe4ugre3sgotaalycnqlueyijwm6ak6h2dxvkkg6aww2vtia`. Notice that this means it doesn't have a `Modified` date, just a `Created` date. It's unnamed, so can't be edited.

## Delete a resource

Deleteing a named resource is easy:

```
% ul delete /foo/hello.txt
% ul ls /foo
Packages URI
bar      ul:bafkreifniakjf4kpctuyqc2hw33qbjuk7f4j6stnyldnkmqg6lyux4fnju#c14n0

```

Deleting an unnamed resource is a little trickier - you need to reference it by its CID identifier.

```
% ul delete /foo/bar/bafkreiadxiqe4ugre3sgotaalycnqlueyijwm6ak6h2dxvkkg6aww2vtia
% ul ls /foo/bar
Assertions URI                                                            Created                   Modified
jd         ul:bafkreigsyouvprcm5wqo7l5zeehitmkiw25gjvrbz5d4pqeowaupw3zzdi 2020-05-05T19:40:06-04:00 2020-05-05T19:40:06-04:00

```

You can also remove packages:

```
% ul delete /foo
% ul ls /
Files          URI                                                                    Created                   Modified                  Format              Size
context.jsonld dweb:/ipfs/bafkreibwyzeetse6wpzntw2rj5jkblxccysntihmezupsjwgx532ttrwxm 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 221
package.jsonld dweb:/ipfs/bafkreif7y43zk6p7hlvmarjveblgiqyy2266wxounchapnakmuxdjqdpam 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 1240
schema.jsonld  dweb:/ipfs/bafkreigpvc6gujkuvja4usyjykcpakdmjtoegehls5rdln4mzxenvgxzgq 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 715
shex.jsonld    dweb:/ipfs/bafkreicqpmn3cyvyr4ioy6link2deaik53nih4tqobee7xmco2xpfiu7gu 2020-05-04T20:08:52-04:00 2020-05-04T20:08:52-04:00 application/ld+json 30939

```
