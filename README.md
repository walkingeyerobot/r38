# r38
R-38 is an insulation strength. For managing drafts.

sqlite3:
```
sqlite> .schema
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE users( id integer primary key autoincrement, google_id text unique, email text, picture text, slack string, discord string);
CREATE TABLE seats( id integer primary key autoincrement, position number, user number, draft number, round number default 1);
CREATE TABLE packs( id integer primary key autoincrement, seat number, modified number, round number , original_seat number);
CREATE TABLE cards( id integer primary key autoincrement, pack number, edition text, number text, tags text, name text, faceup number default false, original_pack number);
CREATE TABLE drafts( id integer primary key autoincrement, name text);
CREATE TABLE revealed( id integer primary key autoincrement, draft number, message text);
CREATE TABLE events( id integer primary key autoincrement, draft number, user number, announcement text, card1 number, card2 number, modified number);
CREATE VIEW v_packs as select packs.*, count(cards.id) as count from packs left join cards on packs.id=cards.pack group by packs.id
/* v_packs(id,seat,modified,round,original_seat,count) */;
```
