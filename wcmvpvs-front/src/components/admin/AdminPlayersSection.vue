<template>
  <section class="card">
    <SectionHeader
      title="Giocatori"
      :description="`Gestisci fino a ${playerSlotCount} giocatori da mostrare nella pagina di voto.`"
    />

    <p v-if="!props.teams.length" class="info-banner">
      Aggiungi almeno una squadra per assegnare correttamente i giocatori
      salvati nel database.
    </p>

    <p v-if="playerOverflow.length" class="info-banner warning">
      Sono presenti {{ playerOverflow.length }} giocatori aggiuntivi nel
      database. Verranno rimossi al prossimo salvataggio.
    </p>

    <div class="player-slots">
      <fieldset
        v-for="(slot, index) in playerSlots"
        :key="`player-slot-${index}`"
        class="player-slot"
      >
        <legend>Giocatore {{ index + 1 }}</legend>
        <div class="player-slot__grid">
          <label>
            Nome
            <input v-model.trim="slot.first_name" type="text" placeholder="Es. Mario" />
          </label>
          <label>
            Cognome
            <input v-model.trim="slot.last_name" type="text" placeholder="Es. Rossi" />
          </label>
          <label>
            Ruolo
            <input v-model.trim="slot.role" type="text" placeholder="Es. Schiacciatore" />
          </label>
          <label>
            Numero di maglia
            <input v-model="slot.jersey_number" type="number" min="0" inputmode="numeric" placeholder="Es. 7" />
          </label>
          <label>
            Squadra
            <select v-model.number="slot.team_id">
              <option :value="0">Seleziona squadra</option>
              <option v-for="team in props.teams" :key="team.id" :value="team.id">
                {{ team.name }}
              </option>
            </select>
          </label>
          <label class="checkbox-inline">
            <input type="checkbox" v-model="slot.is_called_up" />
            <span>Convocato</span>
          </label>
          <label>
            URL immagine (opzionale)
            <input
              v-model.trim="slot.image_url"
              type="url"
              placeholder="https://..."
              @input="handlePlayerUrlChange(index)"
            />
          </label>
          <label class="file-input">
            Oppure carica immagine
            <input type="file" accept="image/*" @change="handlePlayerImageChange(index, $event)" />
          </label>
          <div v-if="slot.image_preview" class="player-slot__preview" aria-label="Anteprima immagine giocatore">
            <img :src="slot.image_preview" alt="Anteprima giocatore" />
            <button class="btn link" type="button" @click="removePlayerImage(index)">Rimuovi</button>
          </div>
        </div>
      </fieldset>
    </div>

    <div class="player-schema">
      <label>
        Schema convocati
        <select v-model.number="rosterSchema">
          <option :value="12">12 giocatori</option>
          <option :value="13">13 giocatori</option>
          <option :value="14">14 giocatori</option>
        </select>
      </label>
      <p class="muted small">
        Il layout della pagina di voto si adatterà automaticamente in base al
        numero di convocati selezionato.
      </p>
    </div>

    <div class="actions-row">
      <button class="btn outline" type="button" @click="restorePlayerSlots" :disabled="isSavingPlayers">
        Ripristina dati salvati
      </button>
      <button class="btn primary" type="button" @click="savePlayers" :disabled="isSavingPlayers">
        {{ isSavingPlayers ? 'Salvataggio…' : 'Salva giocatori' }}
      </button>
    </div>

    <p v-if="playerSaveError" class="error">{{ playerSaveError }}</p>
    <p v-if="playerSaveMessage" class="success-message">{{ playerSaveMessage }}</p>
  </section>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { apiClient } from '../../api';
import SectionHeader from './ui/SectionHeader.vue';
import { DEFAULT_ROSTER_SCHEMA, MAX_PLAYER_SLOTS } from '../../roster';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  isSuperAdmin: { type: Boolean, default: false },
  teams: { type: Array, default: () => [] },
});

const playerSlotCount = MAX_PLAYER_SLOTS;
const PLAYER_IMAGE_MAX_WIDTH = 600;
const PLAYER_IMAGE_MAX_HEIGHT = 600;
const PLAYER_IMAGE_QUALITY = 0.75;

const players = ref([]);
const playerOverflow = ref([]);
const isSavingPlayers = ref(false);
const playerSaveError = ref('');
const playerSaveMessage = ref('');
const rosterSchema = ref(DEFAULT_ROSTER_SCHEMA);

const validateRosterSchema = (value) =>
  value === 12 || value === 13 || value === 14 ? value : DEFAULT_ROSTER_SCHEMA;

const createEmptyPlayerSlot = (teamId = 0) => ({
  id: 0,
  first_name: '',
  last_name: '',
  role: '',
  jersey_number: '',
  team_id: teamId,
  is_called_up: true,
  image_url: '',
  image_preview: '',
  _imageChangeToken: null,
});

const playerSlots = reactive(
  Array.from({ length: playerSlotCount }, () => createEmptyPlayerSlot()),
);

const fallbackTeamId = () => (props.teams.length ? props.teams[0].id : 0);

const resetPlayerSlot = (slot) => {
  Object.assign(slot, createEmptyPlayerSlot(fallbackTeamId()));
};

const ensurePlayerSlotTeams = () => {
  const fallback = fallbackTeamId();
  if (!fallback) return;
  playerSlots.forEach((slot) => {
    if (!slot.team_id) slot.team_id = fallback;
  });
};

const slotHasContent = (slot) => {
  if (!slot) return false;
  const jersey =
    typeof slot.jersey_number === 'number'
      ? slot.jersey_number.toString()
      : `${slot.jersey_number || ''}`;
  return (
    slot.first_name.trim() ||
    slot.last_name.trim() ||
    slot.role.trim() ||
    jersey.trim() ||
    slot.image_url.trim()
  );
};

const normalizePlayerPayload = (slot, fallbackTeam) => {
  const sanitizedJersey = Number(slot.jersey_number);
  const jerseyNumber = Number.isFinite(sanitizedJersey) ? sanitizedJersey : 0;
  return {
    first_name: slot.first_name.trim(),
    last_name: slot.last_name.trim(),
    role: slot.role.trim(),
    jersey_number: jerseyNumber,
    image_url: slot.image_url.trim(),
    team_id: slot.team_id || fallbackTeam || 0,
    is_called_up: Boolean(slot.is_called_up),
  };
};

const normalizePlayerResponse = (item) => {
  const firstName = typeof item?.first_name === 'string' ? item.first_name.trim() : '';
  const lastName = typeof item?.last_name === 'string' ? item.last_name.trim() : '';
  const role = typeof item?.role === 'string' ? item.role.trim() : '';
  const jerseyRaw = typeof item?.jersey_number === 'number' ? item.jersey_number : Number(item?.jersey_number);
  const jerseyNumber = Number.isFinite(jerseyRaw) ? jerseyRaw : 0;
  const image = typeof item?.image_url === 'string' ? item.image_url.trim() : '';
  const team = Number(item?.team_id) || 0;
  const isCalledUp = Boolean(
    item && Object.prototype.hasOwnProperty.call(item, 'is_called_up') ? item.is_called_up : true,
  );
  return {
    id: Number(item?.id) || 0,
    first_name: firstName,
    last_name: lastName,
    role,
    jersey_number: jerseyNumber,
    image_url: image,
    team_id: team,
    is_called_up: isCalledUp,
  };
};

const sortPlayersForDisplay = (a, b) => {
  const jerseyA = a.jersey_number || Number.MAX_SAFE_INTEGER;
  const jerseyB = b.jersey_number || Number.MAX_SAFE_INTEGER;
  if (jerseyA !== jerseyB) return jerseyA - jerseyB;
  const lastComp = a.last_name.localeCompare(b.last_name);
  if (lastComp !== 0) return lastComp;
  const firstComp = a.first_name.localeCompare(b.first_name);
  if (firstComp !== 0) return firstComp;
  return a.id - b.id;
};

const applyPlayersToSlots = () => {
  const sorted = [...players.value].sort(sortPlayersForDisplay);
  players.value = sorted;
  playerOverflow.value = sorted.length > playerSlotCount ? sorted.slice(playerSlotCount) : [];
  const fallback = fallbackTeamId();
  for (let i = 0; i < playerSlotCount; i++) {
    const slot = playerSlots[i];
    const player = sorted[i];
    if (slot && player) {
      Object.assign(slot, {
        id: player.id,
        first_name: player.first_name,
        last_name: player.last_name,
        role: player.role,
        jersey_number: player.jersey_number ? player.jersey_number.toString() : '',
        team_id: player.team_id || fallback,
        is_called_up: player.is_called_up,
        image_url: player.image_url,
        image_preview: player.image_url || '',
      });
    } else if (slot) {
      resetPlayerSlot(slot);
    }
  }
  ensurePlayerSlotTeams();
};

const restorePlayerSlots = () => {
  applyPlayersToSlots();
  playerSaveError.value = '';
  playerSaveMessage.value = '';
};

async function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
    reader.onerror = () => reject(reader.error || new Error('Impossibile leggere il file'));
    reader.readAsDataURL(file);
  });
}

const loadImageFromDataUrl = (dataUrl) =>
  new Promise((resolve, reject) => {
    const image = new Image();
    image.decoding = 'async';
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error('Impossibile caricare l\'immagine selezionata.'));
    image.src = dataUrl;
  });

const toDataUrlSafely = (canvas, type, quality) => {
  try {
    return typeof quality === 'number' ? canvas.toDataURL(type, quality) : canvas.toDataURL(type);
  } catch {
    return '';
  }
};

const extractMimeType = (dataUrl) => {
  if (typeof dataUrl !== 'string') return '';
  const match = /^data:([^;]+);/i.exec(dataUrl);
  return match ? match[1] : '';
};

const optimizePlayerImage = async (file) => {
  const originalDataUrl = await readFileAsDataUrl(file);
  if (!originalDataUrl) return '';
  try {
    const image = await loadImageFromDataUrl(originalDataUrl);
    const { naturalWidth: width, naturalHeight: height } = image;
    if (!width || !height) return originalDataUrl;
    const scale = Math.min(1, PLAYER_IMAGE_MAX_WIDTH / width, PLAYER_IMAGE_MAX_HEIGHT / height);
    const targetWidth = Math.max(1, Math.round(width * scale));
    const targetHeight = Math.max(1, Math.round(height * scale));
    const canvas = document.createElement('canvas');
    canvas.width = targetWidth;
    canvas.height = targetHeight;
    const context = canvas.getContext('2d');
    if (!context) return originalDataUrl;
    context.drawImage(image, 0, 0, targetWidth, targetHeight);
    const originalType = extractMimeType(originalDataUrl);
    const candidateTypes = Array.from(new Set(['image/webp', 'image/jpeg', originalType].filter(Boolean)));
    let bestDataUrl = originalDataUrl;
    let bestSize = originalDataUrl.length;
    candidateTypes.forEach((type) => {
      const quality = type === 'image/png' ? undefined : PLAYER_IMAGE_QUALITY;
      const candidate = toDataUrlSafely(canvas, type, quality);
      if (candidate && candidate.length < bestSize) {
        bestDataUrl = candidate;
        bestSize = candidate.length;
      }
    });
    return bestDataUrl;
  } catch {
    return originalDataUrl;
  }
};

const handlePlayerImageChange = async (index, event) => {
  const slot = playerSlots[index];
  if (!slot) return;
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  const input = event?.target;
  const file = input?.files?.[0];
  if (!file) {
    slot.image_preview = slot.image_url || '';
    return;
  }
  const changeToken = Symbol('player-image-change');
  slot._imageChangeToken = changeToken;
  try {
    const optimizedDataUrl = await optimizePlayerImage(file);
    if (slot._imageChangeToken === changeToken && optimizedDataUrl) {
      slot.image_url = optimizedDataUrl;
      slot.image_preview = optimizedDataUrl;
    }
  } catch {
    // silently ignore
  } finally {
    if (slot._imageChangeToken === changeToken) slot._imageChangeToken = null;
    if (input) input.value = '';
  }
};

const handlePlayerUrlChange = (index) => {
  const slot = playerSlots[index];
  if (!slot) return;
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  slot.image_preview = slot.image_url || '';
};

const removePlayerImage = (index) => {
  const slot = playerSlots[index];
  if (!slot) return;
  playerSaveMessage.value = '';
  playerSaveError.value = '';
  slot.image_url = '';
  slot.image_preview = '';
};

async function loadPlayers() {
  try {
    const { data } = await apiClient.get('/players', props.authHeaders);
    const schemaCandidate = Number(data?.roster_schema);
    if (Number.isFinite(schemaCandidate)) {
      rosterSchema.value = validateRosterSchema(schemaCandidate);
    }
    const payload = Array.isArray(data?.players) ? data.players : data;
    players.value = Array.isArray(payload) ? payload.map(normalizePlayerResponse) : [];
    applyPlayersToSlots();
  } catch (e) {
    console.error('Errore caricamento giocatori', e);
  }
}

async function savePlayers() {
  if (isSavingPlayers.value) return;
  playerSaveError.value = '';
  playerSaveMessage.value = '';
  const fallback = fallbackTeamId();
  const hasAnyContent = playerSlots.some(slotHasContent);
  if (!fallback && hasAnyContent) {
    playerSaveError.value = 'Crea almeno una squadra e assegnala ai giocatori prima di salvare.';
    return;
  }
  isSavingPlayers.value = true;
  const handledIds = new Set();
  try {
    const schemaToSave = validateRosterSchema(rosterSchema.value);
    await apiClient.put('/players/settings', { roster_schema: schemaToSave }, props.authHeaders);

    for (const slot of playerSlots) {
      const hasContent = slotHasContent(slot);
      if (hasContent) {
        const payload = normalizePlayerPayload(slot, fallback);
        if (!payload.first_name || !payload.last_name || !payload.role) {
          playerSaveError.value = 'Nome, cognome e ruolo sono obbligatori per ogni giocatore salvato.';
          isSavingPlayers.value = false;
          return;
        }
        if (!payload.team_id) {
          playerSaveError.value = 'Seleziona una squadra per ogni giocatore salvato.';
          isSavingPlayers.value = false;
          return;
        }
        if (slot.id) {
          await apiClient.put(`/players/${slot.id}`, payload, props.authHeaders);
          handledIds.add(slot.id);
        } else {
          const { data } = await apiClient.post('/players', payload, props.authHeaders);
          const createdId = Number(data?.id) || 0;
          if (createdId) {
            slot.id = createdId;
            handledIds.add(createdId);
          }
        }
      } else if (slot.id) {
        await apiClient.delete(`/players/${slot.id}`, props.authHeaders);
        handledIds.add(slot.id);
        resetPlayerSlot(slot);
      } else {
        resetPlayerSlot(slot);
      }
    }

    for (const player of players.value) {
      if (!handledIds.has(player.id)) {
        await apiClient.delete(`/players/${player.id}`, props.authHeaders);
        handledIds.add(player.id);
      }
    }

    await loadPlayers();
    playerSaveMessage.value = 'Giocatori salvati con successo.';
  } catch (e) {
    if (!playerSaveError.value) {
      playerSaveError.value = 'Si è verificato un errore durante il salvataggio dei giocatori. Riprova.';
    }
  } finally {
    isSavingPlayers.value = false;
  }
}

onMounted(loadPlayers);
</script>
