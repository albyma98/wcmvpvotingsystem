import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { DatabaseSync } from 'node:sqlite'

const dir = 'wcmvpvs-back/service/api/'
const migration = readFileSync(dir + 'club_mvp_migrations/001_mvp_light.sql', 'utf8')
const clubStore = readFileSync(dir + 'club_store.go', 'utf8')
const store = readFileSync(dir + 'club_mvp_store.go', 'utf8')
const open = store.match(/const clubMVPOpenMatches = `([^]*?)` \+ clubMVPRosterJoin \+ `([^]*?)`/).slice(1).join(
  store.match(/const clubMVPRosterJoin = `([^]*?)`/)[1],
)
const joinRoster = store.match(/const clubMVPRosterJoin = `([^]*?)`/)[1]
const insertVote = store.match(/`(INSERT INTO club_mvp_votes\(match_id,player_id,voter_hash\)[^]*?)`\+clubMVPOpenMatches\+`([^]*?)`\+clubMVPRosterJoin\+`([^]*?)`/)
const voteSql = insertVote[1] + open + insertVote[2] + joinRoster + insertVote[3]
for (const foreignKeys of [true, false]) {
  const db = new DatabaseSync(':memory:')
  db.exec('PRAGMA foreign_keys=' + (foreignKeys ? 'ON' : 'OFF'))
  for (const table of ['club_sites', 'club_teams', 'club_players', 'club_matches']) {
    db.exec(clubStore.match(new RegExp('`(CREATE TABLE IF NOT EXISTS ' + table + ' \\([^]*?\\);)`'))[1])
  }
  db.exec(migration)
  db.exec("INSERT INTO club_mvp_cookie_key (id,secret) VALUES (1, randomblob(32))")
  const secret = db.prepare('SELECT hex(secret) AS secret FROM club_mvp_cookie_key').get().secret
  db.exec(migration)
  assert.equal(db.prepare('SELECT hex(secret) AS secret FROM club_mvp_cookie_key').get().secret, secret)
  db.exec(`
    INSERT INTO club_sites (id,organization_id,slug,name) VALUES (1,1,'club','Club');
    INSERT INTO club_teams (id,site_id,name,slug) VALUES (1,1,'U19','u19'), (2,1,'U17','u17');
    INSERT INTO club_players (id,site_id,team_id,first_name) VALUES (1,1,1,'Anna'), (2,1,1,'Maria'), (3,1,2,'Lara');
    INSERT INTO club_matches (id,site_id,team_id,opponent,scheduled_at,status)
      VALUES (1,1,1,'B','now','live'), (2,1,1,'C','now','live');
    INSERT INTO club_mvp_matches (match_id,site_id,enabled) VALUES (1,1,1),(2,1,1);
    INSERT INTO club_mvp_callups (match_id,player_id,first_name,last_name,jersey_number,role,position)
      VALUES (1,1,'Anna','Rossi',7,'',0),(1,3,'Lara','Rossi',8,'',1),(2,2,'Maria','Rossi',9,'',0);
  `)
  assert.equal(db.prepare('SELECT m.id' + open).all(1).length, 2)
  const vote = db.prepare(voteSql)
  const device = Buffer.alloc(32, 1)
  assert.equal(vote.run(2, device, 1, 1, 2).changes, 0)
  assert.equal(vote.run(3, device, 1, 1, 3).changes, 0)
  assert.equal(vote.run(1, device, 1, 1, 1).changes, 1)
  assert.equal(vote.run(1, device, 1, 1, 1).changes, 0)
  assert.equal(vote.run(2, device, 1, 2, 2).changes, 1)
  assert.throws(() => db.exec('UPDATE club_matches SET team_id=2 WHERE id=1'), /mvp_has_votes/)
  assert.throws(() => db.exec('DELETE FROM club_matches WHERE id=1'), /mvp_has_votes/)
  db.exec("UPDATE club_matches SET status='completed' WHERE id=1")
  assert.equal(vote.run(1, Buffer.alloc(32, 2), 1, 1, 1).changes, 0)
  const final = db.prepare('SELECT enabled,closed_at,finalized_at FROM club_mvp_matches WHERE match_id=1').get()
  assert.equal(final.enabled, 0)
  assert.ok(final.closed_at && final.finalized_at)
  db.exec("DELETE FROM club_players WHERE id=1")
  assert.equal(db.prepare('SELECT first_name FROM club_mvp_callups c JOIN club_mvp_votes v USING (match_id,player_id) WHERE match_id=1').get().first_name, 'Anna')
  db.exec('DELETE FROM club_mvp_votes WHERE match_id=1; DELETE FROM club_matches WHERE id=1')
  assert.equal(db.prepare('SELECT COUNT(*) AS count FROM club_mvp_callups WHERE match_id=1').get().count, 0)
  db.close()
  console.log(`SQLite schema and vote statement passed with foreign_keys=${foreignKeys}`)
}
