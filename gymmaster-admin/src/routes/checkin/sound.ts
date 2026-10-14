/** Âm báo ngắn bằng Web Audio API (không cần file âm thanh). */
let ctx: AudioContext | null = null;

function audio(): AudioContext | null {
  try {
    const Ctor =
      window.AudioContext ??
      (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!Ctor) return null;
    ctx ??= new Ctor();
    if (ctx.state === 'suspended') void ctx.resume();
    return ctx;
  } catch {
    return null;
  }
}

function tone(
  ac: AudioContext,
  freq: number,
  start: number,
  duration: number,
  type: OscillatorType,
) {
  const osc = ac.createOscillator();
  const gain = ac.createGain();
  osc.type = type;
  osc.frequency.value = freq;
  const t0 = ac.currentTime + start;
  gain.gain.setValueAtTime(0.0001, t0);
  gain.gain.exponentialRampToValueAtTime(0.25, t0 + 0.01);
  gain.gain.exponentialRampToValueAtTime(0.0001, t0 + duration);
  osc.connect(gain).connect(ac.destination);
  osc.start(t0);
  osc.stop(t0 + duration + 0.02);
}

/** Thành công: hai nốt cao đi lên. */
export function playSuccess() {
  const ac = audio();
  if (!ac) return;
  tone(ac, 880, 0, 0.12, 'sine');
  tone(ac, 1320, 0.13, 0.18, 'sine');
}

/** Từ chối: hai tiếng trầm. */
export function playError() {
  const ac = audio();
  if (!ac) return;
  tone(ac, 220, 0, 0.18, 'square');
  tone(ac, 180, 0.24, 0.26, 'square');
}
