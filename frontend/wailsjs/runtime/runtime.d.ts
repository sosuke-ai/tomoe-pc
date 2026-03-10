export function EventsOn(eventName: string, callback: (...data: any[]) => void): () => void;
export function EventsEmit(eventName: string, ...data: any[]): void;
export function EventsOff(eventName: string, ...additionalEventNames: string[]): void;
export function WindowShow(): void;
export function WindowHide(): void;
export function WindowMinimise(): void;
export function WindowMaximise(): void;
export function WindowSetTitle(title: string): void;
export function Quit(): void;
