import { useEffect, useRef, useState, type PointerEvent } from 'react';
import styles from './WhiteboardCanvas.module.css';

type Point = { x: number; y: number };
type Stroke = { color: string; width: number; points: Point[] };

interface Props {
  data: string;
  editable: boolean;
  onChange: (data: string) => void;
}

function parseStrokes(data: string): Stroke[] {
  try {
    const parsed = JSON.parse(data);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((stroke): stroke is Stroke => Array.isArray(stroke?.points) && stroke.points.length > 0);
  } catch {
    return [];
  }
}

function drawCanvas(canvas: HTMLCanvasElement, strokes: Stroke[]) {
  const context = canvas.getContext('2d');
  if (!context) return;
  const bounds = canvas.getBoundingClientRect();
  const ratio = window.devicePixelRatio || 1;
  canvas.width = Math.max(1, Math.floor(bounds.width * ratio));
  canvas.height = Math.max(1, Math.floor(bounds.height * ratio));
  context.setTransform(ratio, 0, 0, ratio, 0, 0);
  context.clearRect(0, 0, bounds.width, bounds.height);
  context.fillStyle = '#fffdf8';
  context.fillRect(0, 0, bounds.width, bounds.height);
  context.lineCap = 'round';
  context.lineJoin = 'round';
  strokes.forEach((stroke) => {
    if (stroke.points.length < 1) return;
    context.strokeStyle = stroke.color;
    context.lineWidth = stroke.width;
    context.beginPath();
    context.moveTo(stroke.points[0].x, stroke.points[0].y);
    stroke.points.slice(1).forEach((point) => context.lineTo(point.x, point.y));
    context.stroke();
  });
}

export default function WhiteboardCanvas({ data, editable, onChange }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const drawingRef = useRef(false);
  const [strokes, setStrokes] = useState<Stroke[]>(() => parseStrokes(data));
  const [color, setColor] = useState('#165c92');

  useEffect(() => {
    if (!drawingRef.current) setStrokes(parseStrokes(data));
  }, [data]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const redraw = () => drawCanvas(canvas, strokes);
    redraw();
    window.addEventListener('resize', redraw);
    return () => window.removeEventListener('resize', redraw);
  }, [strokes]);

  const pointFromEvent = (event: PointerEvent<HTMLCanvasElement>): Point => {
    const bounds = event.currentTarget.getBoundingClientRect();
    return { x: event.clientX - bounds.left, y: event.clientY - bounds.top };
  };

  const startStroke = (event: PointerEvent<HTMLCanvasElement>) => {
    if (!editable) return;
    drawingRef.current = true;
    event.currentTarget.setPointerCapture(event.pointerId);
    setStrokes((previous) => [...previous, { color, width: 3, points: [pointFromEvent(event)] }]);
  };

  const extendStroke = (event: PointerEvent<HTMLCanvasElement>) => {
    if (!drawingRef.current) return;
    const point = pointFromEvent(event);
    setStrokes((previous) => {
      if (previous.length === 0) return previous;
      const next = [...previous];
      next[next.length - 1] = { ...next[next.length - 1], points: [...next[next.length - 1].points, point] };
      onChange(JSON.stringify(next));
      return next;
    });
  };

  const finishStroke = (event: PointerEvent<HTMLCanvasElement>) => {
    if (!drawingRef.current) return;
    drawingRef.current = false;
    event.currentTarget.releasePointerCapture(event.pointerId);
    onChange(JSON.stringify(strokes));
  };

  const clear = () => {
    if (!editable) return;
    setStrokes([]);
    onChange('[]');
  };

  return (
    <div className={styles.canvasShell}>
      {editable && <div className={styles.canvasTools}>
        <label>Pen <input type="color" value={color} onChange={(event) => setColor(event.target.value)} /></label>
        <button type="button" onClick={clear}>Clear board</button>
      </div>}
      <canvas
        ref={canvasRef}
        className={`${styles.canvas} ${editable ? styles.canvasEditable : ''}`}
        onPointerDown={startStroke}
        onPointerMove={extendStroke}
        onPointerUp={finishStroke}
        onPointerCancel={finishStroke}
        aria-label="Shared whiteboard canvas"
      />
    </div>
  );
}
