import { useCallback, useEffect, useRef } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { Button } from './Button';

interface SignaturePadCanvasProps {
  value?: string;
  onChange: (dataUrl: string | null) => void;
  disabled?: boolean;
  label?: string;
}

const CANVAS_HEIGHT = 160;
const STROKE_WIDTH = 2;
const STROKE_COLOR = '#111827';
const FALLBACK_WIDTH = 300;

// Sizes the drawing buffer for the device pixel ratio so strokes stay sharp
// on high-DPI mobile screens while CSS keeps the layout size.
function setupCanvas(canvas: HTMLCanvasElement, width: number): void {
  const dpr = window.devicePixelRatio || 1;
  canvas.width = Math.round(width * dpr);
  canvas.height = Math.round(CANVAS_HEIGHT * dpr);
  canvas.style.width = `${width}px`;
  canvas.style.height = `${CANVAS_HEIGHT}px`;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  ctx.scale(dpr, dpr);
  ctx.lineWidth = STROKE_WIDTH;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';
  ctx.strokeStyle = STROKE_COLOR;
}

function capturePointer(canvas: HTMLCanvasElement, pointerId: number): void {
  if (typeof canvas.setPointerCapture !== 'function' || typeof pointerId !== 'number') return;
  try {
    canvas.setPointerCapture(pointerId);
  } catch {
    // Synthetic pointer ids (jsdom, test drivers) throw InvalidPointerId;
    // capture only keeps strokes alive outside the canvas, so it is optional.
  }
}

function pointFromEvent(event: ReactPointerEvent<HTMLCanvasElement>) {
  const rect = event.currentTarget.getBoundingClientRect();
  return { x: event.clientX - rect.left, y: event.clientY - rect.top };
}

function useSignatureStrokes(
  onChange: (dataUrl: string | null) => void,
  disabled: boolean,
  value?: string,
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const isDrawingRef = useRef(false);
  const hasInkRef = useRef(false);

  const getCtx = useCallback((): CanvasRenderingContext2D | null => {
    return canvasRef.current?.getContext('2d') ?? null;
  }, []);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    setupCanvas(canvas, containerRef.current?.clientWidth || FALLBACK_WIDTH);
  }, []);

  const clearCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    const ctx = getCtx();
    if (!canvas || !ctx) return;
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    hasInkRef.current = false;
  }, [getCtx]);

  // A parent-side reset (value cleared) wipes the drawing so the pad and the
  // form state cannot diverge.
  useEffect(() => {
    if (!value) clearCanvas();
  }, [value, clearCanvas]);

  function onPointerDown(event: ReactPointerEvent<HTMLCanvasElement>) {
    if (disabled) return;
    const ctx = getCtx();
    if (!ctx) return;
    capturePointer(event.currentTarget, event.pointerId);
    const { x, y } = pointFromEvent(event);
    ctx.beginPath();
    ctx.moveTo(x, y);
    isDrawingRef.current = true;
  }

  function onPointerMove(event: ReactPointerEvent<HTMLCanvasElement>) {
    if (disabled || !isDrawingRef.current) return;
    const ctx = getCtx();
    if (!ctx) return;
    const { x, y } = pointFromEvent(event);
    ctx.lineTo(x, y);
    ctx.stroke();
    hasInkRef.current = true;
  }

  function endStroke() {
    if (!isDrawingRef.current) return;
    isDrawingRef.current = false;
    const canvas = canvasRef.current;
    if (canvas && hasInkRef.current) {
      onChange(canvas.toDataURL('image/png'));
    }
  }

  function clear() {
    clearCanvas();
    onChange(null);
  }

  return { containerRef, canvasRef, onPointerDown, onPointerMove, endStroke, clear };
}

export function SignaturePadCanvas({
  value,
  onChange,
  disabled = false,
  label = 'Assinatura',
}: SignaturePadCanvasProps) {
  const { containerRef, canvasRef, onPointerDown, onPointerMove, endStroke, clear } =
    useSignatureStrokes(onChange, disabled, value);

  return (
    <div>
      <span className="block text-sm font-medium text-gray-700">{label}</span>
      <div
        ref={containerRef}
        className={`mt-1 overflow-hidden rounded-lg border border-gray-300 bg-white ${
          disabled ? 'opacity-50' : ''
        }`}
      >
        <canvas
          ref={canvasRef}
          role="img"
          aria-label={label}
          className="block h-40 w-full touch-none"
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={endStroke}
          onPointerCancel={endStroke}
        />
      </div>
      <div className="mt-1 flex justify-end">
        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={clear}
          disabled={disabled}
        >
          Limpar
        </Button>
      </div>
    </div>
  );
}
