<script setup>
import { ref } from 'vue';

import AppMenuItem from './AppMenuItem.vue';

const model = ref([
    {
        label: 'Dashboard',
        items: [{ label: 'Panoramica', icon: 'pi pi-fw pi-home', to: '/admin/master' }]
    },
    {
        label: 'Gestione eventi',
        items: [
            { label: 'Eventi', icon: 'pi pi-fw pi-calendar', to: '/admin/events' },
            { label: 'Shop', icon: 'pi pi-fw pi-shopping-bag', to: '/admin/shop' },
            { label: 'Sponsor', icon: 'pi pi-fw pi-megaphone', to: '/admin/sponsors' }
        ]
    },
    {
        label: 'Operazioni',
        items: [
            { label: 'Lotteria', icon: 'pi pi-fw pi-gift', to: '/admin/lottery' },
            { label: 'Validazione biglietti', icon: 'pi pi-fw pi-id-card', to: '/admin/validation' }
        ]
    }
]);
</script>

<template>
    <ul class="layout-menu">
        <template v-for="(item, i) in model" :key="item">
            <app-menu-item v-if="!item.separator" :item="item" :index="i"></app-menu-item>
            <li v-if="item.separator" class="menu-separator"></li>
        </template>
    </ul>
</template>

<style lang="scss" scoped></style>
