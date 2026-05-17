export class PlayerSlots {
  private readonly occupied = new Set<number>();

  constructor(private readonly maxPlayers: number) {}

  assign(): number | null {
    for (let player = 1; player <= this.maxPlayers; player += 1) {
      if (!this.occupied.has(player)) {
        this.occupied.add(player);
        return player;
      }
    }

    return null;
  }

  release(player: number): void {
    this.occupied.delete(player);
  }
}
