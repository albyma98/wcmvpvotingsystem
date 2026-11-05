<script>
import { useController } from '../../../components/ReportAnalysisView.vue?admin-v2-controller';

export default {name:'V2ReportAnalysisView',components:{},props:{analysis:{type:Object,required:true},kpis:{type:Array,default:()=>[]},status:String},emits:[],setup(props,{emit}) {return useController(props,emit);}};
</script>
<template>
  <section class="v2-report-analysis" aria-label="Analisi del report">
    <p class="v2-report-analysis__source">{{ status === 'ready' ? 'Analisi OpenAI validata' : 'Sintesi deterministica' }}</p>
    <h3>{{ analysis.headline }}</h3><p>{{ analysis.summary }}</p>
    <template v-for="(items, kind) in { 'Evidenze positive': analysis.positive_insights, 'Criticità e attenzione': analysis.critical_insights }" :key="kind"><div v-if="items?.length"><h4>{{ kind }}</h4><article v-for="(item, index) in items" :key="index"><strong>{{ item.title }}</strong><p>{{ item.description }}</p><small>{{ analysisMetricReferences(item, kpis) }}</small></article></div></template>
    <div v-if="analysis.recommendations?.length"><h4>Indicazioni operative</h4><article v-for="(item, index) in analysis.recommendations" :key="index"><strong>{{ item.action }}</strong><p>{{ item.reason }}</p><small>Priorità {{ { high: 'alta', medium: 'media', low: 'bassa' }[item.priority] }} · {{ analysisMetricReferences(item, kpis) }}</small></article></div>
    <details v-if="analysis.limitations?.length"><summary>Limiti dell’analisi</summary><ul><li v-for="(limit, index) in analysis.limitations" :key="index">{{ limit }}</li></ul></details>
  </section>
</template>
