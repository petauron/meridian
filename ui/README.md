# Meridian application UI

This directory owns Meridian's page composition and its independently signed
browser bundle. Center provides authenticated data, platform actions, and the
shared component library; it does not compile Meridian pages into its own web
application after the migration.

The build uses the reviewed Vastora UI source at commit
`270e92cc90a4d095dd4db245ec9e83bd7101688a`. Set `VASTORA_WEB_SOURCE` to
that checkout's `web/src` directory. The build fails if the checkout points to
a different commit. `src/index.tsx` exports the versioned `mount`/`update`/
`unmount` entry used by Center. The output filename is tied to this package
version and is published as a TUF target with the official app catalog.
`npm run check` validates the same pinned dependency and type-checks the
application-owned pages against its platform contract before publication. It
creates an ignored `.center-src` link to the reviewed checkout; CI uses the
same preparation step for both catalog build passes.

Center remains responsible for authentication and authorization on every API
request. Loading the UI bundle does not grant new permissions to an app or a
third-party catalog source.
