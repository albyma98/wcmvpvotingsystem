import { onBeforeUnmount, reactive, ref } from 'vue';
export function useFeedback() {
 const pending = reactive(new Set()); const toasts = ref([]); const confirmation = reactive({ open: false, title: '', message: '', pending: false }); let callback; const timers = new Set();
 function notify(message, tone = 'success') {
  const id = Date.now() + Math.random(); toasts.value.push({ id, message, tone });
  const timer = setTimeout(() => { toasts.value = toasts.value.filter(t => t.id !== id); timers.delete(timer); }, 6500); timers.add(timer);
 }
 async function run(key, fn, success) {
  if (pending.has(key)) return; pending.add(key);
  try { const result = await fn(); if (success) notify(success); return result; }
  catch (e) { notify(e?.response?.data?.message || e?.message || 'Operazione non riuscita. Riprova.', 'error'); }
  finally { pending.delete(key); }
 }
 function confirmAction(title, message, action) { callback = action; Object.assign(confirmation, { open: true, title, message, pending: false }); }
 async function acceptConfirmation() { if (confirmation.pending) return; confirmation.pending = true; try { await callback?.(); confirmation.open = false; } catch(e) { notify(e.message, 'error'); } finally { confirmation.pending = false; } }
 onBeforeUnmount(() => timers.forEach(clearTimeout));
 return { run, busy: key => pending.has(key), notify, toasts, confirmation, confirmAction, acceptConfirmation };
}
