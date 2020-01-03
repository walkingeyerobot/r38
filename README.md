# r38
R-38 is an insulation strength. For managing drafts.

sqlite3:
```
sqlite> .schema
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE users( id integer primary key autoincrement, google_id text unique, email text, picture text );
CREATE TABLE seats( id integer primary key autoincrement, position number, user number, draft number );
CREATE TABLE packs( id integer primary key autoincrement, seat number, modified number, round number );
CREATE TABLE cards( id integer primary key autoincrement, pack number, edition text, number text, tags text, name text );
CREATE TABLE drafts( id integer primary key autoincrement, name text, round number default 1);
```
