<template>
  <svg viewBox="0 0 100 100" fill="none" aria-hidden="true" focusable="false">
    <circle cx="50" cy="50" r="46" fill="#e68032" />
    <circle cx="50" cy="50" r="41" fill="#f29946" />
    <path d="M20 15C41-1 75 8 88 30 65 16 38 18 13 40c0-10 2-17 7-25Z" fill="#ffbd70" opacity=".45" />
    <path d="M91 29C92 68 59 95 24 88c23 16 54 7 66-15 8-14 8-30 1-44Z" fill="#8e3c15" opacity=".5" />
    <g fill="#7c3b1b" opacity=".2">
      <g v-for="row in 6" :key="row">
        <circle v-for="dot in 6" :key="dot" :cx="24 + dot * 7 + (row % 2) * 3" :cy="23 + row * 8" r=".7" />
      </g>
    </g>
    <g stroke="#542a19" stroke-width="2.3" stroke-linecap="round">
      <path d="M8 31c28 23 52 28 84 38M65 7C42 31 37 60 35 93" />
      <path d="M29 9c22 25 28 58 41 82M5 54c25-16 55-16 89-12" />
    </g>
    <circle cx="50" cy="50" r="45.2" stroke="#ffbc76" stroke-width="1.6" />
    <path d="M19 30c6-9 13-14 23-16" stroke="#ffe0aa" stroke-width="4" stroke-linecap="round" opacity=".55" />
  </svg>
</template>
