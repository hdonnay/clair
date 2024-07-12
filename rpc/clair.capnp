@0x97651196d36545f0;
using Go = import "/go.capnp";
$Go.package("proto");
$Go.import("github.com/quay/clair/v4/rpc/internal/proto");

# Common/helper parts:

# Map is a generic map type.
struct Map(Key, Value) {
  entries @0 :List(Entry);
  struct Entry {
    key @0 :Key;
    value @1 :Value;
  }
}

# Iterator is a generic iterator.
#
# The function signature looks odd, but think of it as a "push" iterator; it's driven server-side.
# The caller must call "done" to check errors.
interface Iterator(T) {
	next @0 (item :T) -> stream;
	done @1 ();
}

# Digest is the standard OCI digest type, with the exception that the actual digest is unencoded.
struct Digest {
	algorithm @0 :Text;
	digest @1 :Data;
}

interface Main {
	capabilities @0 () -> (avail :List(Available));
	indexer @1 () -> (srv :Indexer);
	matcher @2 () -> (srv :Matcher);

	enum Available {
		indexer @0;
		matcher @1;
	}
}

# Indexer parts:

interface Indexer {
	submit @0 (manifest :Manifest) -> (meta :Metadata);
	getMeta @1 (manifest :Digest) -> (meta :Metadata);
	getIndex @2(manifest :Digest) -> (meta :Metadata, index :Index);
}

interface Index {
	environments @0 (cb :Iterator(Environment)) -> ();
	packages @1 (cb :Iterator(Package)) -> ();
	attrsFor @2 (pkg :Text, cb: Iterator(Attr)) -> ();
}

struct Metadata {
	digest @0 :Digest;
	state @1 :Text;
	error :union {
		unset @2 :Void;
		ok @3 :Void;
		error @4 :Text;
	}
}

struct Manifest {
	digest @0 :Digest;
	layers @1 :List(Layer);
	annotations @2 :Annotations;

	struct Layer {
		digest @0 :Digest;
		uri @1 :Text;
		mediaType @2 :Text;
		headers @3 :Headers;
		annotations @4 :Annotations;
	}

	using Headers = Map(Text, List(Text));
	using Annotations = Map(Text, Text);
}


struct Environment {
	id @0 :Text;
}

struct Package {
	id @0 :Text;
}

struct Attr {
	packageId @0 :Text;
	kind @1 :Text;
	data @2 :Map(Text, Text);
}

# Matcher:

interface ReportCallback {
	environment @0 (env :Environment) -> stream;
	package @1 (pkg :Package) -> stream;
	done @2 ();
}

interface Matcher {
	match @0 (index :Index, report :ReportCallback) -> ();
}
