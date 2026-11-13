import { inject, onUnmounted, provide } from 'vue';
import { createSponsorAnalyticsService, sponsorAnalyticsKey } from './sponsorAnalyticsService.js';
import { createSponsorMeasurement } from './sponsorMeasurement.js';

export function provideSponsorTracking({ url, scope, matchday = false, device }) {
  const service = createSponsorAnalyticsService({ url, device });
  const measurement = createSponsorMeasurement({ service, scope, measureGeometryViewability: matchday });
  provide(sponsorAnalyticsKey, measurement);
  onUnmounted(measurement.close);
  return measurement;
}

export function useSponsorPlacement(measurement = inject(sponsorAnalyticsKey, null)) {
  const update = (element, binding) => measurement?.track(element, binding.value);
  return { mounted: update, updated: update, beforeUnmount: (element) => measurement?.track(element, null) };
}
