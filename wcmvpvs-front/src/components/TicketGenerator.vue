<template>
  <div class="container center-align">
    <h4>Genera Biglietto</h4>
    <button class="btn waves-effect" @click="generateTicket">Genera</button>
    <div v-if="ticket.code" class="ticket-info">
      <p>Codice: {{ ticket.code }}</p>
      <p>Firma: {{ ticket.signature }}</p>
      <img :src="qrUrl" alt="QR code" />
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from "vue";

const ticket = reactive({ code: "", signature: "", qr_data: "" });
const qrUrl = ref("");

async function generateTicket() {
  const res = await fetch("http://localhost:3000/ticket", {
    method: "POST",
  });
  const data = await res.json();
  ticket.code = data.code;
  ticket.signature = data.signature;
  ticket.qr_data = data.qr_data;
  qrUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(ticket.qr_data)}`;
}
</script>

<style scoped>
.ticket-info {
  margin-top: 1rem;
}
</style>