<template>
  <section class="card">
    <header class="section-header">
      <h2>Sponsor</h2>
      <p>Gestisci fino a {{ maxSponsors }} sponsor da mostrare nella schermata pubblica.</p>
    </header>

    <div class="sponsor-controls" role="group" aria-label="Visibilità sponsor">
      <label class="sponsor-range">
        <span>Numero di sponsor visibili: {{ desiredActiveSponsorCount }} / {{ maxSponsors }}</span>
        <input
          type="range"
          min="0"
          :max="sponsorSliderMax"
          v-model.number="desiredActiveSponsorCount"
          @change="applyActiveSponsorCount"
          :disabled="!sponsors.length || isApplyingSponsorCount"
        />
      </label>
      <p class="muted small">Gli sponsor attivi vengono mostrati nell'ordine indicato qui sotto.</p>
    </div>

    <form @submit.prevent="createSponsor" class="form-grid sponsor-form">
      <label>
        Nome sponsor
        <input v-model.trim="newSponsor.name" type="text" placeholder="Es. Partner ufficiale" />
      </label>
      <label>
        Nome report
        <input v-model.trim="newSponsor.reportName" type="text" placeholder="Etichetta per i report (es. Sponsor A)" />
      </label>
      <label>
        Link (opzionale)
        <input v-model.trim="newSponsor.linkUrl" type="url" placeholder="https://example.com" />
      </label>
      <label class="file-input">
        Logo sponsor
        <input type="file" accept="image/*" @change="handleNewSponsorLogoChange" />
      </label>
      <div v-if="newSponsor.logoData" class="sponsor-preview new" aria-label="Anteprima logo nuovo sponsor">
        <img :src="newSponsor.logoData" alt="Anteprima logo sponsor" />
      </div>
      <button class="btn primary" type="submit" :disabled="isCreatingSponsor">
        {{ isCreatingSponsor ? 'Salvataggio…' : 'Aggiungi sponsor' }}
      </button>
    </form>

    <p v-if="sponsorError" class="error">{{ sponsorError }}</p>

    <ul v-if="sponsors.length" class="item-list sponsors-list">
      <li v-for="sponsor in sponsors" :key="sponsor.id" class="item sponsor-item">
        <div class="item-body sponsor-body">
          <div class="sponsor-preview" :aria-label="`Logo sponsor ${sponsor.name || sponsor.position}`">
            <img v-if="sponsor.logoData" :src="sponsor.logoData" :alt="`Logo ${sponsor.name || 'sponsor'}`" />
            <span v-else class="empty-logo">Logo non disponibile</span>
          </div>
          <div class="sponsor-fields">
            <div class="form-grid compact">
              <label>
                Nome sponsor
                <input v-model.trim="sponsor.name" type="text" />
              </label>
              <label>
                Nome report
                <input v-model.trim="sponsor.reportName" type="text" placeholder="Etichetta interna" />
              </label>
              <label>
                Link (opzionale)
                <input v-model.trim="sponsor.linkUrl" type="url" placeholder="https://example.com" />
              </label>
              <label class="file-input">
                Aggiorna logo
                <input type="file" accept="image/*" @change="(event) => handleSponsorLogoChange(event, sponsor)" />
              </label>
            </div>
            <p class="muted sponsor-meta">
              Posizione {{ sponsor.position }} • {{ sponsor.isActive ? 'Visibile' : 'Nascosto' }}
            </p>
          </div>
        </div>
        <div class="item-actions vertical">
          <button
            class="btn secondary"
            type="button"
            @click="updateSponsorEntry(sponsor)"
            :disabled="sponsorBeingUpdated === sponsor.id"
          >
            <span v-if="sponsorBeingUpdated === sponsor.id">Salvataggio…</span>
            <span v-else>Salva</span>
          </button>
          <button
            class="btn danger"
            type="button"
            @click="deleteSponsorEntry(sponsor.id)"
            :disabled="sponsorBeingDeleted === sponsor.id"
          >
            <span v-if="sponsorBeingDeleted === sponsor.id">Eliminazione…</span>
            <span v-else>Elimina</span>
          </button>
        </div>
      </li>
    </ul>
    <p v-else class="muted text-center">Nessuno sponsor configurato al momento.</p>
  </section>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { apiClient } from '../../api';

const props = defineProps({
  authHeaders: { type: Object, required: true },
  isSuperAdmin: { type: Boolean, default: false },
});

const maxSponsors = 4;
const sponsors = ref([]);
const sponsorError = ref('');
const isCreatingSponsor = ref(false);
const sponsorBeingUpdated = ref(0);
const sponsorBeingDeleted = ref(0);
const isApplyingSponsorCount = ref(false);
const desiredActiveSponsorCount = ref(0);

const newSponsor = reactive({
  name: '',
  reportName: '',
  linkUrl: '',
  logoData: '',
  isActive: true,
});

const activeSponsorCount = computed(() => sponsors.value.filter((s) => s.isActive).length);
const sponsorSliderMax = computed(() =>
  sponsors.value.length ? Math.min(maxSponsors, sponsors.value.length) : maxSponsors,
);

function normalizeSponsorResponse(item) {
  if (!item || typeof item !== 'object') return null;
  return {
    id: Number(item.id) || 0,
    name: typeof item.name === 'string' ? item.name.trim() : '',
    reportName: typeof item.report_name === 'string' ? item.report_name.trim() : (typeof item.reportName === 'string' ? item.reportName.trim() : ''),
    linkUrl: typeof item.link_url === 'string' ? item.link_url.trim() : '',
    position: Number(item.position) || 0,
    logoData: typeof item.logo_data === 'string' ? item.logo_data : '',
    isActive: Boolean(item.is_active),
  };
}

function serializeSponsorPayload(sponsor) {
  return {
    name: sponsor.name.trim(),
    report_name: (sponsor.reportName || '').trim(),
    link_url: sponsor.linkUrl.trim(),
    position: sponsor.position,
    logo_data: sponsor.logoData,
    is_active: sponsor.isActive,
  };
}

function sortedSponsors() {
  return [...sponsors.value].sort((a, b) => a.position - b.position);
}

function nextSponsorPosition() {
  const used = new Set(sponsors.value.map((s) => s.position));
  for (let i = 1; i <= maxSponsors; i++) {
    if (!used.has(i)) return i;
  }
  return Math.min(maxSponsors, sponsors.value.length + 1);
}

function recomputeActiveSponsorSlider() {
  desiredActiveSponsorCount.value = Math.min(sponsorSliderMax.value, activeSponsorCount.value);
}

function resetNewSponsorForm() {
  Object.assign(newSponsor, { name: '', reportName: '', linkUrl: '', logoData: '', isActive: true });
}

async function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
    reader.onerror = () => reject(reader.error || new Error('Impossibile leggere il file'));
    reader.readAsDataURL(file);
  });
}

async function handleSponsorLogoChange(event, targetSponsor) {
  const [file] = event?.target?.files || [];
  if (!file) return;
  sponsorError.value = '';
  try {
    const dataUrl = await readFileAsDataUrl(file);
    if (dataUrl) targetSponsor.logoData = dataUrl;
  } catch (e) {
    sponsorError.value = 'Impossibile caricare il logo selezionato.';
  } finally {
    if (event?.target) event.target.value = '';
  }
}

async function handleNewSponsorLogoChange(event) {
  await handleSponsorLogoChange(event, newSponsor);
}

async function loadSponsors() {
  try {
    const { data } = await apiClient.get('/admin/sponsors', props.authHeaders);
    sponsors.value = Array.isArray(data)
      ? data.map(normalizeSponsorResponse).filter((s) => s && s.id).sort((a, b) => a.position - b.position)
      : [];
    recomputeActiveSponsorSlider();
  } catch (e) {
    console.error('Errore caricamento sponsor', e);
  }
}

async function createSponsor() {
  if (isCreatingSponsor.value) return;
  sponsorError.value = '';
  if (sponsors.value.length >= maxSponsors) {
    sponsorError.value = `Puoi configurare al massimo ${maxSponsors} sponsor.`;
    return;
  }
  if (!newSponsor.logoData) {
    sponsorError.value = 'Carica un logo per lo sponsor.';
    return;
  }
  const payload = serializeSponsorPayload({
    ...newSponsor,
    name: newSponsor.name.trim(),
    position: nextSponsorPosition(),
    isActive: false,
  });
  isCreatingSponsor.value = true;
  try {
    await apiClient.post('/admin/sponsors', payload, props.authHeaders);
    resetNewSponsorForm();
    await loadSponsors();
  } catch (e) {
    if (e?.response?.status === 400) {
      sponsorError.value = 'Controlla i dati inseriti: sono disponibili massimo 4 sponsor.';
    }
  } finally {
    isCreatingSponsor.value = false;
  }
}

async function updateSponsorEntry(sponsor) {
  if (sponsorBeingUpdated.value === sponsor.id) return;
  sponsorError.value = '';
  if (!sponsor.logoData) {
    sponsorError.value = 'Carica un logo per lo sponsor.';
    return;
  }
  sponsorBeingUpdated.value = sponsor.id;
  try {
    const payload = serializeSponsorPayload({ ...sponsor, name: sponsor.name.trim() });
    await apiClient.put(`/admin/sponsors/${sponsor.id}`, payload, props.authHeaders);
    await loadSponsors();
  } catch (e) {
    if (e?.response?.status === 400) sponsorError.value = 'Controlla i dati dello sponsor e riprova.';
    else if (e?.response?.status === 404) sponsorError.value = 'Sponsor non trovato. Aggiorna la pagina.';
  } finally {
    sponsorBeingUpdated.value = 0;
  }
}

async function deleteSponsorEntry(id) {
  if (sponsorBeingDeleted.value === id) return;
  sponsorError.value = '';
  sponsorBeingDeleted.value = id;
  try {
    await apiClient.delete(`/admin/sponsors/${id}`, props.authHeaders);
    await loadSponsors();
  } catch (e) {
    if (e?.response?.status === 404) sponsorError.value = 'Sponsor già rimosso.';
  } finally {
    sponsorBeingDeleted.value = 0;
  }
}

async function applyActiveSponsorCount() {
  if (isApplyingSponsorCount.value || !sponsors.value.length) {
    desiredActiveSponsorCount.value = 0;
    return;
  }
  sponsorError.value = '';
  const target = Math.max(0, Math.min(maxSponsors, desiredActiveSponsorCount.value));
  isApplyingSponsorCount.value = true;
  try {
    const updates = [];
    sortedSponsors().forEach((sponsor, index) => {
      const shouldBeActive = index < target;
      if (sponsor.isActive !== shouldBeActive) {
        const payload = serializeSponsorPayload({ ...sponsor, isActive: shouldBeActive });
        updates.push(apiClient.put(`/admin/sponsors/${sponsor.id}`, payload, props.authHeaders));
      }
    });
    if (updates.length) await Promise.all(updates);
    await loadSponsors();
  } catch (e) {
    if (e?.response?.status === 400) {
      sponsorError.value = 'Impossibile aggiornare il numero di sponsor visibili.';
    }
  } finally {
    isApplyingSponsorCount.value = false;
  }
}

onMounted(loadSponsors);
</script>
