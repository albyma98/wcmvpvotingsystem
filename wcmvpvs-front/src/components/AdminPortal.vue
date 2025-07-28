<template>
  <div>
    <h3>Admin Portal</h3>
    <div>
      <button class="btn" @click="section = 'teams'">Teams</button>
      <button class="btn" @click="section = 'players'">Players</button>
      <button class="btn" @click="section = 'events'">Events</button>
      <button class="btn" @click="section = 'admins'">Admins</button>
    </div>
    <div v-if="section === 'teams'">
      <h4>Teams</h4>
      <form @submit.prevent="createTeam">
        <input v-model="newTeam" placeholder="Team name" />
        <button type="submit" class="btn">Add</button>
      </form>
      <ul>
        <li v-for="t in teams" :key="t.id">
          {{ t.name }}
          <button class="btn" @click="deleteTeam(t.id)">Del</button>
        </li>
      </ul>
    </div>
    <div v-if="section === 'players'">
      <h4>Players</h4>
       <form @submit.prevent="createPlayer">
        <input v-model="player.first_name" placeholder="First name" />
        <input v-model="player.last_name" placeholder="Last name" />
        <input v-model="player.role" placeholder="Role" />
        <input v-model.number="player.jersey_number" placeholder="Number" />
        <input v-model="player.image_url" placeholder="Image URL" />
        <select v-model.number="player.team_id">
          <option disabled value="0">Select Team</option>
          <option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
        <button type="submit" class="btn">Add</button>
      </form>
      <ul>
        <li v-for="p in players" :key="p.id">
          {{ p.first_name }} {{ p.last_name }}
          <button class="btn" @click="deletePlayer(p.id)">Del</button>
        </li>
      </ul>
    </div>
    <div v-if="section === 'events'">
      <h4>Events</h4>
        <form @submit.prevent="createEvent">
        <select v-model.number="event.team1_id">
          <option disabled value="0">Select Home Team</option>
          <option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
        <select v-model.number="event.team2_id">
          <option disabled value="0">Select Away Team</option>
          <option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
        <input type="date" v-model="event.start_datetime" />
        <input v-model="event.location" placeholder="Location" />
        <button type="submit" class="btn">Add</button>
      </form>
      <ul>
        <li v-for="e in events" :key="e.id">
          {{ teamName(e.team1_id) }} vs {{ teamName(e.team2_id) }} - {{ e.start_datetime }}
          <span class="vote-link">{{ baseLink }}?eventID={{ e.id }}</span>
          <a href="#" @click.prevent="openVote(e.id)">Vote</a>
          <button class="btn" @click="deleteEvent(e.id)">Del</button>
        </li>
      </ul>
    </div>
    <div v-if="section === 'admins'">
      <h4>Admins</h4>
      <form @submit.prevent="createAdmin">
        <input v-model="admin.username" placeholder="Username" />
        <input v-model="admin.password_hash" placeholder="Password" />
        <input v-model="admin.role" placeholder="Role" />
        <button type="submit" class="btn">Add</button>
      </form>
      <ul>
        <li v-for="a in admins" :key="a.id">
          {{ a.username }} - {{ a.role }}
          <button class="btn" @click="deleteAdmin(a.id)">Del</button>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from "vue";
import axios from "axios";

const api = axios.create({ baseURL: "http://localhost:3000" });
const emit = defineEmits(["vote-event"]);

const section = ref("teams");
const teams = ref([]);
const newTeam = ref("");
const players = ref([]);
const player = reactive({
  first_name: "",
  last_name: "",
  role: "",
  jersey_number: 0,
  image_url: "",
  team_id: 0,
});
const events = ref([]);
const event = reactive({
  team1_id: 0,
  team2_id: 0,
  start_datetime: "",
  location: "",
});
const admins = ref([]);
const admin = reactive({ username: "", password_hash: "", role: "" });
const baseLink = window.location.origin + window.location.pathname;
async function loadAll() {
  teams.value = (await api.get("/teams")).data;
  players.value = (await api.get("/players")).data;
  events.value = (await api.get("/events")).data;
  admins.value = (await api.get("/admins")).data;
  await nextTick();
  if (window.M && window.M.FormSelect) {
    const elems = document.querySelectorAll("select");
    window.M.FormSelect.init(elems);
  }
}

onMounted(loadAll);

async function createTeam() {
  await api.post("/teams", { name: newTeam.value });
  newTeam.value = "";
  loadAll();
}
async function deleteTeam(id) {
  await api.delete(`/teams/${id}`);
  loadAll();
}

async function createPlayer() {
  await api.post("/players", player);
  player.first_name = "";
  player.last_name = "";
  player.role = "";
  player.jersey_number = 0;
  player.image_url = "";
  player.team_id = 0;
  loadAll();
}
async function deletePlayer(id) {
  await api.delete(`/players/${id}`);
  loadAll();
}

async function createEvent() {
  await api.post("/events", event);
  event.team1_id = 0;
  event.team2_id = 0;
  event.start_datetime = "";
  event.location = "";
  loadAll();
}
async function deleteEvent(id) {
  await api.delete(`/events/${id}`);
  loadAll();
}
function teamName(id) {
  const t = teams.value.find((tm) => tm.id === id);
  return t ? t.name : "";
}

function openVote(id) {
  const url = `${baseLink}?eventID=${id}`;
  window.history.replaceState({}, "", url);
  emit("vote-event", id);
}
async function createAdmin() {
  await api.post("/admins", admin);
  admin.username = "";
  admin.password_hash = "";
  admin.role = "";
  loadAll();
}
async function deleteAdmin(id) {
  await api.delete(`/admins/${id}`);
  loadAll();
}
</script>

<style scoped>
.btn {
  margin: 0.2rem;
}
input {
  margin: 0.2rem;
}
.vote-link {
  display: block;
  font-size: 0.8rem;
  color: #555;
  margin: 0.2rem 0;
}
</style>