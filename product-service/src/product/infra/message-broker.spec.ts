import { Test, TestingModule } from '@nestjs/testing';
import { MessageBroker } from './message-broker';

describe('MessageBroker', () => {
  let provider: MessageBroker;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [MessageBroker],
    }).compile();

    provider = module.get<MessageBroker>(MessageBroker);
  });

  it('should be defined', () => {
    expect(provider).toBeDefined();
  });
});
