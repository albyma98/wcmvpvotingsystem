import test from 'node:test';
import assert from 'node:assert/strict';
import {createDemoActivityAdapter} from './demoAdapter.js';
import {dayInRome,CATALOG,TREASURE_SPOTS} from './model.js';
import {organizationCrowd} from '../organizationCrowd.js';
const storage=()=>{const values=new Map();return{getItem:k=>values.get(k)||null,setItem:(k,v)=>values.set(k,v)}};
test('daily claim, purchase and equipment persist with no duplicate payments or negative balances',async()=>{
 const disk=storage();let adapter=createDemoActivityAdapter(disk,{dev:true,now:()=>new Date('2026-10-20T12:00:00Z')});let s=await adapter.getState();const claim={action:'claim',day:s.treasure.day,location:s.treasure.location};await adapter.command(claim);s=await adapter.command(claim);assert.equal(s.wallet,20);
 await adapter.command({action:'buy',key:'club-shirt'});s=await adapter.command({action:'buy',key:'club-shirt'});assert.equal(s.wallet,0);await assert.rejects(adapter.command({action:'buy',key:'away-shirt'}));
 await adapter.command({action:'equip',key:'club-shirt'});adapter=createDemoActivityAdapter(disk,{dev:true,now:()=>new Date('2026-10-20T12:00:00Z')});s=await adapter.getState();assert.equal(s.equipment.shirt,'club-shirt');assert.equal(s.wallet,0);assert.equal(s.treasure.claimed,true);
 s=await adapter.debug('day');assert.equal(s.treasure.claimed,false);await assert.rejects(adapter.command(claim));
});
test('prediction cutoff and settlement are idempotent and affect ABX points only',async()=>{
 const adapter=createDemoActivityAdapter(storage(),{dev:true});let s=await adapter.getState();await adapter.command({action:'predict',score:'3-1',event_id:s.matches[0].id});s=await adapter.debug('result','3-1');assert.equal(s.points,50);assert.equal(s.wallet,0);s=await adapter.debug('result','3-1');assert.equal(s.points,50);await assert.rejects(adapter.command({action:'predict',score:'3-2'}));
});
test('debug disabled outside development; demo friends keep other seats and valid inventory',async()=>{
 const disk=storage();const publicAdapter=createDemoActivityAdapter(disk);await assert.rejects(publicAdapter.debug('coins'));const adapter=createDemoActivityAdapter(disk,{dev:true});const a=organizationCrowd((await adapter.getFans()).fans,7);await adapter.debug('friend');const b=organizationCrowd((await adapter.getFans()).fans,7);assert.equal(b.actors.length,a.actors.length+1);assert.deepEqual(a.actors.map(a=>a.seat.id),b.actors.slice(0,a.actors.length).map(a=>a.seat.id));assert.equal(new Set(b.actors.map(a=>a.seat.id)).size,b.actors.length);assert.equal(CATALOG.length,12);assert.equal(TREASURE_SPOTS.length,10);
});
test('Rome daily rollover uses local daylight-saving offset',()=>{assert.equal(dayInRome(new Date('2026-03-29T22:30:00Z')),'2026-03-30');assert.equal(dayInRome(new Date('2026-10-25T23:30:00Z')),'2026-10-26')});
