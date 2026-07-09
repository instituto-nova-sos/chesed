import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { SignaturePadCanvas } from '../SignaturePadCanvas';

const STUB_DATA_URL = 'data:image/png;base64,STUB';

// jsdom ships no canvas 2D implementation, so the pad is tested against its
// contract: pointer strokes must end in an onChange(dataUrl) emission and
// "Limpar" must clear and emit null. The 2D context is a recorded stub.
function createCtxStub() {
  return {
    scale: vi.fn(),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    stroke: vi.fn(),
    clearRect: vi.fn(),
    lineWidth: 0,
    lineCap: 'butt',
    lineJoin: 'miter',
    strokeStyle: '',
  };
}

let ctxStub: ReturnType<typeof createCtxStub>;

beforeEach(() => {
  ctxStub = createCtxStub();
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
    ctxStub as unknown as CanvasRenderingContext2D,
  );
  vi.spyOn(HTMLCanvasElement.prototype, 'toDataURL').mockReturnValue(STUB_DATA_URL);
});

afterEach(() => {
  vi.restoreAllMocks();
});

function drawStroke(canvas: HTMLElement) {
  fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
  fireEvent.pointerMove(canvas, { pointerId: 1, clientX: 40, clientY: 30 });
  fireEvent.pointerUp(canvas, { pointerId: 1, clientX: 40, clientY: 30 });
}

function getCanvas(): HTMLElement {
  return screen.getByRole('img', { name: 'Assinatura' });
}

describe('SignaturePadCanvas', () => {
  it('emits a PNG data URL when a stroke ends', () => {
    const onChange = vi.fn();
    render(<SignaturePadCanvas onChange={onChange} />);

    drawStroke(getCanvas());

    expect(ctxStub.stroke).toHaveBeenCalled();
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith(STUB_DATA_URL);
  });

  it('clears the drawing and emits null on "Limpar"', async () => {
    const onChange = vi.fn();
    render(<SignaturePadCanvas onChange={onChange} />);

    drawStroke(getCanvas());
    const clearCallsBefore = ctxStub.clearRect.mock.calls.length;
    await userEvent.click(screen.getByRole('button', { name: 'Limpar' }));

    expect(ctxStub.clearRect.mock.calls.length).toBeGreaterThan(clearCallsBefore);
    expect(onChange).toHaveBeenLastCalledWith(null);
  });

  it('ignores pointer events when disabled', () => {
    const onChange = vi.fn();
    render(<SignaturePadCanvas onChange={onChange} disabled />);

    drawStroke(getCanvas());

    expect(ctxStub.stroke).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it('does not emit anything for a tap without movement', () => {
    const onChange = vi.fn();
    render(<SignaturePadCanvas onChange={onChange} />);
    const canvas = getCanvas();

    fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
    fireEvent.pointerUp(canvas, { pointerId: 1, clientX: 10, clientY: 10 });

    expect(onChange).not.toHaveBeenCalled();
  });

  it('uses the custom label as the accessible name', () => {
    render(<SignaturePadCanvas onChange={vi.fn()} label="Assinatura do responsável" />);
    expect(screen.getByRole('img', { name: 'Assinatura do responsável' })).toBeInTheDocument();
  });
});
