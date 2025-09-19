import { Test, TestingModule } from '@nestjs/testing';
import { Repositories } from './repositories';

describe('Repositories', () => {
  let provider: Repositories;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [Repositories],
    }).compile();

    provider = module.get<Repositories>(Repositories);
  });

  it('should be defined', () => {
    expect(provider).toBeDefined();
  });
});
